package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/awsutil"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pottava/aws-s3-proxy/internal/config"
)

// // S3get returns a specified object from Amazon S3
func (c *client) S3get(bucket, key string, rangeHeader *string) (*s3.GetObjectOutput, error) {
	req := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Range:  rangeHeader,
	}
	return c.S3.GetObjectWithContext(c.Context, req)
}

// S3listObjects returns a list of s3 objects
func (c *client) S3listObjects(bucket, prefix string) (*s3.ListObjectsOutput, error) {
	req := &s3.ListObjectsInput{
		Bucket:    aws.String(bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String("/"),
	}
	// List 1000 records
	if !config.Config.AllPagesInDir {
		return c.S3.ListObjectsWithContext(c.Context, req)
	}
	// List all objects with pagenation
	result := &s3.ListObjectsOutput{
		CommonPrefixes: []*s3.CommonPrefix{},
		Contents:       []*s3.Object{},
		Prefix:         aws.String(prefix),
	}
	//
	err := c.S3.ListObjectsPagesWithContext(c.Context, req,
		func(page *s3.ListObjectsOutput, lastPage bool) bool {
			result.CommonPrefixes = append(result.CommonPrefixes, page.CommonPrefixes...)
			result.Contents = append(result.Contents, page.Contents...)
			return len(page.Contents) == 1000
		})
	return result, err
}

//S3 Parallel download a object
func (c *client) S3Download(fp http.ResponseWriter, bucket, key string, rangeHeader *string) error {
	req := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Range:  rangeHeader,
	}
	d := &downloader{Context: c.Context,
		cfg:   c,
		in:    req,
		w:     fp,
		index: 1,
		count: 0,
	}

	d.partBodyMaxRetries = 3
	d.totalBytes = -1
	if d.cfg.Concurrency == 0 {
		d.cfg.Concurrency = DefaultDownloadConcurrency
	}

	if d.cfg.PartSize == 0 {
		d.cfg.PartSize = DefaultDownloadPartSize
	}

	_, err := d.download()

	return err
}

type downloader struct {
	context.Context
	cfg                *client
	in                 *s3.GetObjectInput
	w                  io.Writer
	partBodyMaxRetries int
	pos                int64
	totalBytes         int64
	written            int64
	wg                 sync.WaitGroup
	m                  sync.Mutex
	index              int64
	count              int64
	err                error
}

func (d *downloader) download() (n int64, err error) {
	// If range is specified fall back to single download of that range
	// this enables the functionality of ranged gets with the downloader but
	// at the cost of no multipart downloads.
	if rng := aws.StringValue(d.in.Range); len(rng) > 0 {
		d.downloadRange(rng)
		return d.written, d.err
	}

	// Spin off first worker to check additional header information
	d.getChunk()

	if total := d.getTotalBytes(); total >= 0 {
		// Spin up workers
		ch := make(chan dlchunk, d.cfg.Concurrency)

		for i := 0; i < d.cfg.Concurrency; i++ {
			d.wg.Add(1)
			go d.downloadPart(ch)
		}

		// Assign work

		for d.getErr() == nil {
			if d.pos >= total {
				break // We're finished queuing chunks
			}
			atomic.AddInt64(&d.count, 1)
			// Queue the next range of bytes to read.
			ch <- dlchunk{w: d.w, start: d.pos, size: d.cfg.PartSize, num: d.count}
			d.pos += d.cfg.PartSize

		}

		// Wait for completion
		close(ch)
		d.wg.Wait()
	} else {
		// Checking if we read anything new
		for d.err == nil {
			d.getChunk()
		}

		// We expect a 416 error letting us know we are done downloading the
		// total bytes. Since we do not know the content's length, this will
		// keep grabbing chunks of data until the range of bytes specified in
		// the request is out of range of the content. Once, this happens, a
		// 416 should occur.
		e, ok := d.err.(awserr.RequestFailure)
		if ok && e.StatusCode() == http.StatusRequestedRangeNotSatisfiable {
			d.err = nil
		}

	}

	// Return error
	return d.written, d.err
}

func (d *downloader) tryDownloadChunk(in *s3.GetObjectInput, w *dlchunk) (int64, error) {
	resp, err := d.cfg.S3.GetObjectWithContext(d.Context, in)
	if err != nil {
		return 0, err
	}
	d.setTotalBytes(resp) // Set total if not yet set.
	var n int64

	if d.index == w.num {
		n, err = io.Copy(w, resp.Body)
		resp.Body.Close()
		if err != nil {
			return n, err
		}
		atomic.AddInt64(&d.index, 1)
		return n, nil
	}

	for d.index != w.num {
		// wait before index data writen.
		time.Sleep(1 * time.Microsecond)
		if d.index == w.num {
			n, err = io.Copy(w, resp.Body)
			resp.Body.Close()
			if err != nil {
				return n, err
			}
			atomic.AddInt64(&d.index, 1)
			break
		}
	}

	return n, nil
}

func (d *downloader) downloadPart(ch chan dlchunk) {
	defer d.wg.Done()

	for {
		chunk, ok := <-ch
		if !ok {
			break
		}
		if d.getErr() != nil {
			// Drain the channel if there is an error, to prevent deadlocking
			// of download producer.
			continue
		}
		if err := d.downloadChunk(chunk); err != nil {
			d.setErr(err)
		}
	}
}

func (d *downloader) downloadRange(rng string) {
	if d.getErr() != nil {
		return
	}
	atomic.AddInt64(&d.count, 1)
	chunk := dlchunk{w: d.w, start: d.pos, num: d.count}
	// Ranges specified will short circuit the multipart download
	chunk.withRange = rng

	if err := d.downloadChunk(chunk); err != nil {
		d.setErr(err)
	}

	// Update the position based on the amount of data received.
	d.pos = d.written
}

// downloadChunk downloads the chunk from s3
func (d *downloader) downloadChunk(chunk dlchunk) error {
	in := &s3.GetObjectInput{}
	awsutil.Copy(in, d.in)

	// Get the next byte range of data
	in.Range = aws.String(chunk.ByteRange())

	var n int64
	var err error
	for retry := 0; retry <= d.partBodyMaxRetries; retry++ {
		n, err = d.tryDownloadChunk(in, &chunk)
		if err == nil {
			break
		}
		// Check if the returned error is an errReadingBody.
		// If err is errReadingBody this indicates that an error
		// occurred while copying the http response body.
		// If this occurs we unwrap the err to set the underlying error
		// and attempt any remaining retries.
		if bodyErr, ok := err.(*errReadingBody); ok {
			err = bodyErr.Unwrap()
		} else {
			return err
		}

		chunk.cur = 0
		// logMessage(d.cfg.S3, aws.LogDebugWithRequestRetries,
		// 	fmt.Sprintf("DEBUG: object part body download interrupted %s, err, %v, retrying attempt %d",
		// 		aws.StringValue(in.Key), err, retry))
	}

	d.incrWritten(n)

	return err
}

// getChunk grabs a chunk of data from the body.
// Not thread safe. Should only used when grabbing data on a single thread.
func (d *downloader) getChunk() {
	if d.getErr() != nil {
		return
	}

	atomic.AddInt64(&d.count, 1)
	chunk := dlchunk{w: d.w, start: d.pos, size: d.cfg.PartSize, num: d.count}
	d.pos += d.cfg.PartSize

	if err := d.downloadChunk(chunk); err != nil {
		d.setErr(err)
	}
}

// getErr is a thread-safe getter for the error object
func (d *downloader) getErr() error {
	d.m.Lock()
	defer d.m.Unlock()

	return d.err
}

// setErr is a thread-safe setter for the error object
func (d *downloader) setErr(e error) {
	d.m.Lock()
	defer d.m.Unlock()

	d.err = e
}

// getTotalBytes is a thread-safe getter for retrieving the total byte status.
func (d *downloader) getTotalBytes() int64 {
	d.m.Lock()
	defer d.m.Unlock()

	return d.totalBytes
}
func (d *downloader) setTotalBytes(resp *s3.GetObjectOutput) {
	d.m.Lock()
	defer d.m.Unlock()

	if d.totalBytes >= 0 {
		return
	}

	if resp.ContentRange == nil {

		// ContentRange is nil when the full file contents is provided, and
		// is not chunked. Use ContentLength instead.
		if resp.ContentLength != nil {
			d.totalBytes = *resp.ContentLength
			return
		}
	} else {
		parts := strings.Split(*resp.ContentRange, "/")

		total := int64(-1)
		var err error
		// Checking for whether or not a numbered total exists
		// If one does not exist, we will assume the total to be -1, undefined,
		// and sequentially download each chunk until hitting a 416 error
		totalStr := parts[len(parts)-1]
		if totalStr != "*" {
			total, err = strconv.ParseInt(totalStr, 10, 64)
			if err != nil {
				d.err = err
				return
			}
		}

		d.totalBytes = total
	}
}

func (d *downloader) incrWritten(n int64) {
	d.m.Lock()
	defer d.m.Unlock()

	d.written += n
}

type dlchunk struct {
	w     io.Writer
	start int64
	size  int64
	cur   int64
	num   int64
	// specifies the byte range the chunk should be downloaded with.
	withRange string
}

func (c *dlchunk) Write(p []byte) (n int, err error) {
	if c.cur >= c.size && len(c.withRange) == 0 {
		return 0, io.EOF
	}

	n, err = c.w.Write(p)
	c.cur += int64(n)

	return
}

// ByteRange returns a HTTP Byte-Range header value that should be used by the
// client to request the chunk's range.
func (c *dlchunk) ByteRange() string {
	if len(c.withRange) != 0 {
		return c.withRange
	}

	return fmt.Sprintf("bytes=%d-%d", c.start, c.start+c.size-1)
}

type errReadingBody struct {
	err error
}

func (e *errReadingBody) Error() string {
	return fmt.Sprintf("failed to read part body: %v", e.err)
}

func (e *errReadingBody) Unwrap() error {
	return e.err
}

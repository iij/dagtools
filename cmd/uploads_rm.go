package cmd

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/iij/dagtools/client"
	"github.com/iij/dagtools/env"
)

type uploadsRmCommand struct {
	env       *env.Environment
	cli       client.StorageClient
	opts      *flag.FlagSet
	recursive bool
}

func (c *uploadsRmCommand) Description() string {
	return "delete multipart-upload[s]"
}

func (c *uploadsRmCommand) Usage() string {
	return fmt.Sprintf(`Command Usage:
  uploads rm [-r] <bucket>:<key>[:<uploadId>] ...

Options:
%s`, OptionUsage(c.opts))
}

func (c *uploadsRmCommand) Init(env *env.Environment) (err error) {
	c.env = env
	c.cli, _ = client.NewStorageClient(env)
	c.opts = flag.NewFlagSet("uploads rm", flag.ExitOnError)
	c.opts.Usage = func() {
		fmt.Fprintln(os.Stdout, c.Usage())
	}
	c.opts.BoolVar(&c.recursive, "r", false, "recursively list subdirectories encountered")
	return
}

func (c *uploadsRmCommand) Run(args []string) (err error) {
	var (
		bucket   string
		key      string
		uploadId string
		slice    []string
	)
	c.opts.Parse(args)
	for _, arg := range c.opts.Args() {
		slice = strings.Split(arg, ":")
		bucket = slice[0]
		if len(slice) > 2 {
			_uid := slice[len(slice)-1]
			if len(_uid) == 108 {
				uploadId = _uid
			}
		}
		if len(slice) > 1 {
			key = strings.Join(slice[1:], ":")
			if uploadId != "" {
				key = strings.TrimSuffix(key, ":"+uploadId)
			}
		}
		if uploadId == "" {
			listing, err := c.cli.ListMultipartUploads(bucket, key, "", "", "", 1000)
			if err != nil {
				return err
			}
			if len(listing.Uploads) == 0 {
				return errors.New("no such upload")
			}
			c.removeUploads(listing)
		} else {
			// specify uploadId
			upload := client.MultipartUpload{
				Bucket:   bucket,
				Key:      key,
				UploadID: uploadId,
			}
			err = c.cli.AbortMultipartUpload(&upload)
			if err != nil {
				return err
			}
			if c.env.Verbose {
				fmt.Printf("delete: %s%s (ID: %s)\n", bucket, key, uploadId)
			}
		}
	}
	return
}

func (c *uploadsRmCommand) removeUploads(listing *client.MultipartUploadListing) error {
	for _, upload := range listing.Uploads {
		if !c.recursive && upload.Key != listing.Prefix {
			continue
		}
		err := c.cli.AbortMultipartUpload(&client.MultipartUpload{
			Bucket:   listing.Bucket,
			Key:      upload.Key,
			UploadID: upload.UploadId,
		})
		if c.env.Verbose {
			fmt.Printf("delete: %s:%s (ID: %s)\n", listing.Bucket, upload.Key, upload.UploadId)
		}
		if err != nil {
			return err
		}
	}
	if listing.IsTruncated {
		nextListing, err := c.cli.NextListMultipartUploads(listing)
		if err != nil {
			return err
		}
		c.removeUploads(nextListing)
	}
	return nil
}

func init() {
	uploadsSubCommands["rm"] = new(uploadsRmCommand)
}

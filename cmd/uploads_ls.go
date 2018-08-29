package cmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/iij/dagtools/client"
	"github.com/iij/dagtools/env"
)

type uploadsLsCommand struct {
	env        *env.Environment
	cli        client.StorageClient
	opts       *flag.FlagSet
	recursive  bool
	outputJSON bool
}

func (c *uploadsLsCommand) Description() string {
	return "list uploads"
}

func (c *uploadsLsCommand) Usage() string {
	return fmt.Sprintf(`Command Usage:
  uploads ls <bucket>[:<prefix>] ...
  uploads ls -r <bucket>[:<prefix>]
  uploads ls -json <bucket>[:<prefix>]

Options:
%v`, OptionUsage(c.opts))
}

func (c *uploadsLsCommand) Init(env *env.Environment) (err error) {
	c.env = env
	c.cli, _ = client.NewStorageClient(env)
	c.opts = flag.NewFlagSet("uploads ls", flag.ExitOnError)
	c.opts.Usage = func() {
		fmt.Fprintln(os.Stdout, c.Usage())
	}
	c.opts.BoolVar(&c.recursive, "r", false, "recursively list subdirectories encountered")
	c.opts.BoolVar(&c.outputJSON, "json", false, "JSON output")
	return
}

func (c *uploadsLsCommand) Run(args []string) (err error) {
	var (
		bucket string
		key    string
		slice  []string
	)
	c.opts.Parse(args)
	args = c.opts.Args()
	if len(args) == 0 {
		return ErrArgument
	}
	for _, arg := range args {
		slice = strings.Split(arg, ":")
		bucket = slice[0]
		if len(slice) > 1 {
			key = strings.Join(slice[1:], ":")
		}
		listing, err := c.cli.ListMultipartUploads(bucket, key, "", "", "/", 1000)
		if err != nil {
			return err
		}
		if c.outputJSON {
			if err := c.printJSON(listing, true, true); err != nil {
				return err
			}
		} else {
			if err := c.printUploads(listing, true); err != nil {
				return err
			}
		}
	}
	return
}

func (c *uploadsLsCommand) printHeader(listing *client.MultipartUploadListing) {
	fmt.Printf("[%s:%s]\n", listing.Bucket, listing.Prefix)
	fmt.Printf("%20s  %20s  name (upload ID)\n", "owner", "initiated")
}

func (c *uploadsLsCommand) printUploads(listing *client.MultipartUploadListing, head bool) error {
	if head && ! listing.IsEmpty() {
		c.printHeader(listing)
	}
	for _, prefix := range listing.CommonPrefixes {
		if strings.HasPrefix(prefix, listing.Prefix) {
			prefix = prefix[len(listing.Prefix):]
		}
		fmt.Printf("%20s  %20s  %s\n", "-", "-", prefix)
	}
	for _, summary := range listing.Uploads {
		owner := summary.Owner.DisplayName
		if len(owner) > 20 {
			owner = owner[0:19] + "$"
		}
		key := summary.Key
		if strings.HasPrefix(key, listing.Prefix) {
			key = key[len(listing.Prefix):]
		}
		fmt.Printf("%20s  %20s  %s (%s)\n", owner, LocalTimeString(summary.Initiated), key, summary.UploadId)
	}
	if listing.IsTruncated {
		nextListing, err := c.cli.NextListMultipartUploads(listing)
		if err != nil {
			return err
		}
		c.printUploads(nextListing, false)
	}
	if c.recursive {
		for _, cm := range listing.CommonPrefixes {
			yaListing, err := c.cli.ListMultipartUploads(listing.Bucket, cm, "", "", "/", 1000)
			if err != nil {
				return err
			}
			fmt.Println("")
			c.printUploads(yaListing, true)
		}
	}
	return nil
}

func (c *uploadsLsCommand) printJSON(listing *client.MultipartUploadListing, root bool, head bool) error {
	if head {
		fmt.Print("[")
	}
	for _, prefix := range listing.CommonPrefixes {
		if ! head {
			fmt.Print(",\n")
		}
		if bs, err := json.Marshal(prefix); err == nil {
			fmt.Printf(`{"Prefix": %s, "Upload": null}`, string(bs))
			head = false
		}
	}
	for _, upload := range listing.Uploads {
		if ! head {
			fmt.Print(",\n")
		}
		if bs, err := json.Marshal(upload); err == nil {
			fmt.Printf(`{"Prefix": null, "Upload": %s}`, string(bs))
			head = false
		}
	}
	if listing.IsTruncated {
		nextListing, err := c.cli.NextListMultipartUploads(listing)
		if err != nil {
			return err
		}
		c.printJSON(nextListing, false, head)
	}
	if c.recursive {
		for _, prefix := range listing.CommonPrefixes {
			yaListing, err := c.cli.ListMultipartUploads(listing.Bucket, prefix, "", "", "/", 1000)
			if err != nil {
				return err
			}
			c.printJSON(yaListing, false, head)
		}
	}
	if root {
		fmt.Println("]")
	}
	return nil
}

func init() {
	uploadsSubCommands["ls"] = new(uploadsLsCommand)
}

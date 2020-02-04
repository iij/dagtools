package cmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/iij/dagtools/client"
	"github.com/iij/dagtools/env"
)

type lsCommand struct {
	env           *env.Environment
	cli           client.StorageClient
	opts          *flag.FlagSet
	recursive     bool
	humanReadable bool
	outputTSV     bool
	outputJSON    bool
	includeETag   bool
}

func (c *lsCommand) Description() string {
	return "list buckets or objects"
}

func (c *lsCommand) Usage() string {
	return fmt.Sprintf(`Command Usage:
  ls [-r] [-h] [-tsv|-json] [-etag]  [<bucket>[:<file|dir>] ...]

Options:
%s`, OptionUsage(c.opts))
}

func (c *lsCommand) Init(env *env.Environment) (err error) {
	c.env = env
	c.cli, _ = client.NewStorageClient(env)
	c.recursive = false
	opts := flag.NewFlagSet("ls", flag.ExitOnError)
	opts.BoolVar(&c.recursive, "r", false, "recursively list subdirectories encountered")
	opts.BoolVar(&c.humanReadable, "h", false, "Human-readable output. Use unit suffix(B, KB, MB...) for sizes")
	opts.BoolVar(&c.outputTSV, "tsv", false, "TSV output")
	opts.BoolVar(&c.outputJSON, "json", false, "JSON output")
	opts.BoolVar(&c.includeETag, "etag", false, "include ETag")
	opts.Usage = func() {
		fmt.Fprintln(os.Stdout, c.Usage())
	}
	c.opts = opts
	return
}

func (c *lsCommand) Run(args []string) (err error) {
	c.opts.Parse(args)
	var (
		bucket = ""
		prefix = ""
		count  = 0
	)
	argv := c.opts.Args()
	if len(argv) == 0 {
		return c.listBuckets()
	}
	for _, arg := range argv {
		slice := strings.Split(arg, ":")
		bucket = slice[0]
		if len(slice) > 1 {
			prefix = strings.Join(slice[1:], ":")
		}
		if err = c.exec(bucket, prefix, true); err != nil {
			return err
		}
		count++
	}
	return
}

func (c *lsCommand) exec(bucket, prefix string, recursive bool) (err error) {
	num, err := c.listObjects(bucket, prefix, true)
	if err != nil {
		return err
	}
	if num == 0 && strings.HasPrefix(prefix, "/") {
		return c.exec(bucket, strings.TrimLeft(prefix, "/"), recursive)
	}
	return
}

func (c *lsCommand) listBuckets() (err error) {
	listing, err := c.cli.ListBuckets()
	if err != nil {
		return err
	}
	if len(listing.Buckets) == 0 {
		return
	}
	if c.outputTSV {
		c.printBucketsTSV(listing)
	} else if c.outputJSON {
		c.printBucketsJSON(listing)
	} else {
		c.printBuckets(listing)
	}
	return
}

func (c *lsCommand) printBuckets(listing *client.BucketListing) {
	owner := listing.Owner.String()
	if len(owner) > 20 {
		owner = owner[0:20]
	}
	fmt.Fprintf(os.Stdout, "%20s\t%20s\t%s\n", "owner", "created", "name")
	for _, b := range listing.Buckets {
		fmt.Fprintf(os.Stdout, "%20s\t%20s\t%s\n", owner, LocalTimeString(b.CreationDate), b.Name)
	}
}

func (c *lsCommand) printBucketsTSV(listing *client.BucketListing) {
	owner := listing.Owner.String()
	fmt.Fprintf(os.Stdout, "%s\t%s\t%s\n", "owner", "created", "name")
	for _, b := range listing.Buckets {
		fmt.Fprintf(os.Stdout, "%s\t%s\t%s\n", owner, LocalTimeString(b.CreationDate), b.Name)
	}
}

func (c *lsCommand) printBucketsJSON(listing *client.BucketListing) {
	fmt.Print("[")
	for i, b := range listing.Buckets {
		if i != 0 {
			fmt.Print(",\n")
		}
		if bs, err := json.Marshal(b); err == nil {
			fmt.Print(string(bs))
		}
	}
	fmt.Print("]\n")
}

func (c *lsCommand) listObjects(bucket string, prefix string, head bool) (num int, err error) {
	var listing *client.ObjectListing
	_prefix := prefix
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		exists, _, err := c.cli.DoesObjectExist(bucket, prefix)
		if err != nil {
			return -1, err
		}
		if exists {
			listing, err = c.cli.ListObjects(bucket, prefix, "", "", 1)
			if err != nil {
				return -1, err
			}
		}
	}
	if listing == nil {
		if prefix != "" && !(strings.HasSuffix(prefix, "/") || strings.HasSuffix(prefix, "*")) {
			prefix += "/"
		}
		if strings.HasSuffix(prefix, "*") {
			r, _ := regexp.Compile(`\*$`)
			prefix = r.ReplaceAllString(prefix, "")
		}
		listing, err = c.cli.ListObjects(bucket, prefix, "", "/", 1000)
		if err != nil {
			return -1, err
		}
	}
	if listing == nil || listing.IsEmpty() {
		return -1, fmt.Errorf(`no such file or directory: "%s:%s"`, bucket, _prefix)
	}
	if c.outputTSV {
		c.printObjectsTSV(listing, true)
	} else if c.outputJSON {
		c.printObjectsJSON(listing, true, true)
	} else {
		return c.printObjects(listing, true, true)
	}
	return
}

func (c *lsCommand) printObjectsHeader(listing *client.ObjectListing) {
	fmt.Fprintf(os.Stdout, "[%s:%s]\n", listing.Name, listing.Prefix)
	fmt.Fprintf(os.Stdout, "%20s  %16s  %20s", "owner", "size", "last-modified")
	if c.includeETag {
		fmt.Fprintf(os.Stdout, "  %38s", "etag")
	}
	fmt.Fprintf(os.Stdout, "   %s\n", "name")
}

func (c *lsCommand) printObjects(listing *client.ObjectListing, head bool, root bool) (num int, err error) {
	if head && ! listing.IsEmpty() {
		c.printObjectsHeader(listing)
	}
	if listing.CommonPrefixes != nil && len(listing.CommonPrefixes) > 0 {
		for _, p := range listing.CommonPrefixes {
			xs := strings.Split(p.Prefix, "/")
			dirname := xs[len(xs)-2]
			if dirname != "." {
				fmt.Fprintf(os.Stdout, "%20s  %16s  %20s", "-", "-", "-")
				if c.includeETag {
					fmt.Fprintf(os.Stdout, "  %38s", "-")
				}
				fmt.Fprintf(os.Stdout, "   %s/\n", dirname)
				head = false
				num += 1
			}
		}
	}
	if listing.Summaries != nil && len(listing.Summaries) > 0 {
		for _, s := range listing.Summaries {
			owner := s.Owner.String()
			if len(owner) > 20 {
				owner = owner[0:20]
			}
			size := strconv.FormatInt(s.Size, 10)
			if c.humanReadable {
				size = HumanReadableBytes(uint64(s.Size))
			}
			lastMod := LocalTimeString(s.LastModified)
			xs := strings.Split(s.Key, "/")
			filename := xs[len(xs)-1]
			if filename == "" {
				filename = fmt.Sprintf(". -> %s", s.Key)
			}
			fmt.Fprintf(os.Stdout, "%20s  %16s  %20s", owner, size, lastMod)
			if c.includeETag {
				fmt.Fprintf(os.Stdout, "  %38s", s.ETag)
			}
			fmt.Fprintf(os.Stdout, "   %s\n", filename)
			num += 1
			head = false
		}
	}
	if listing.IsTruncated {
		nextListing, err := c.cli.NextListObjects(listing)
		if err != nil {
			return -1, err
		}
		_num, err := c.printObjects(nextListing, head, false)
		if err != nil {
			return -1, err
		}
		num += _num
	}
	if c.recursive {
		for _, cp := range listing.CommonPrefixes {
			yaListing, err := c.cli.ListObjects(listing.Name, cp.Prefix, "", listing.Delimiter, listing.MaxKeys)
			if err != nil {
				return -1, err
			}
			fmt.Println("")
			_num, err := c.printObjects(yaListing, true, false)
			if err != nil {
				return -1, err
			}
			num += _num
		}
	}
	if root {
		fmt.Println("")
	}
	return
}

func (c *lsCommand) printObjectsHeaderTSV(listing *client.ObjectListing) {
	fmt.Fprintf(os.Stdout, "%s\t%s\t%s", "owner", "size", "last-modified")
	if c.includeETag {
		fmt.Fprintf(os.Stdout, "\t%s", "etag")
	}
	fmt.Fprintf(os.Stdout, "\t%s\t%s\n", "bucket", "key")
}

func (c *lsCommand) printObjectsTSV(listing *client.ObjectListing, head bool) (num int, err error) {
	if head && ! listing.IsEmpty() {
		c.printObjectsHeaderTSV(listing)
	}
	if listing.CommonPrefixes != nil && len(listing.CommonPrefixes) > 0 {
		for _, p := range listing.CommonPrefixes {
			fmt.Fprintf(os.Stdout, "%s\t%s\t%s", "-", "-", "-")
			if c.includeETag {
				fmt.Fprintf(os.Stdout, "\t%s", "-")
			}
			fmt.Fprintf(os.Stdout, "\t%s\t%s\t%s\n", listing.Name, listing.Location,  p.Prefix)
			head = false
			num += 1
		}
	}
	if listing.Summaries != nil && len(listing.Summaries) > 0 {
		for _, s := range listing.Summaries {
			owner := s.Owner.String()
			size := strconv.FormatInt(s.Size, 10)
			if c.humanReadable {
				size = HumanReadableBytes(uint64(s.Size))
			}
			lastMod := LocalTimeString(s.LastModified)
			fmt.Fprintf(os.Stdout, "%s\t%s\t%s", owner, size, lastMod)
			if c.includeETag {
				fmt.Fprintf(os.Stdout, "\t%s", s.ETag)
			}
			fmt.Fprintf(os.Stdout, "\t%s\t%s\t%s\n", listing.Name, listing.Location,  s.Key)
			head = false
			num += 1
		}
	}
	if listing.IsTruncated {
		nextListing, err := c.cli.NextListObjects(listing)
		if err != nil {
			return -1, err
		}
		_num, err := c.printObjectsTSV(nextListing, head)
		if err != nil {
			return -1, err
		}
		num += _num
	}
	if c.recursive {
		for _, cp := range listing.CommonPrefixes {
			yaListing, err := c.cli.ListObjects(listing.Name, cp.Prefix, "", listing.Delimiter, listing.MaxKeys)
			if err != nil {
				return -1, err
			}
			_num, err := c.printObjectsTSV(yaListing, true)
			if err != nil {
				return -1, err
			}
			num += _num
		}
	}
	return
}

func (c *lsCommand) printObjectsJSON(listing *client.ObjectListing, head bool, root bool) (num int, err error) {
	if root {
		fmt.Print("[")
	}
	if listing.CommonPrefixes != nil && len(listing.CommonPrefixes) > 0 {
		for _, p := range listing.CommonPrefixes {
			if ! head {
				fmt.Print(",\n")
			}
			if bs, err := json.Marshal(p.Prefix); err == nil {
				fmt.Printf(`{"Prefix": %s, "Content": null}`, string(bs))
				head = false
				num += 1
			}
		}
	}
	if listing.Summaries != nil && len(listing.Summaries) > 0 {
		for _, s := range listing.Summaries {
			if ! head {
				fmt.Print(",\n")
			}
			if bs, err := json.Marshal(s); err == nil {
				fmt.Printf(`{"Prefix": null, "Content": %s}`, string(bs))
				head = false
				num += 1
			}
		}
	}
	if listing.IsTruncated {
		nextListing, err := c.cli.NextListObjects(listing)
		if err != nil {
			return -1, err
		}
		_num, err := c.printObjectsJSON(nextListing, head, false)
		if err != nil {
			return -1, err
		}
		num += _num
	}
	if c.recursive {
		for _, cp := range listing.CommonPrefixes {
			yaListing, err := c.cli.ListObjects(listing.Name, cp.Prefix, "", listing.Delimiter, listing.MaxKeys)
			if err != nil {
				return -1, err
			}
			_num, err := c.printObjectsJSON(yaListing, head, false)
			if err != nil {
				return -1, err
			}
			num += _num
		}
	}
	if root {
		fmt.Print("]\n")
	}
	return
}

func init() {
	Commands.Register(new(lsCommand), "ls")
}

package cmd

import (
	"flag"
	"fmt"
	"github.com/iij/dagtools/client"
	"github.com/iij/dagtools/env"
	"os"
	"strings"
)

type cpCommand struct {
	env           *env.Environment
	cli           client.StorageClient
	opts		  *flag.FlagSet
	recursive	  bool
}

func (c *cpCommand) Description() string {
	return "copy object"
}

func (c *cpCommand) Usage() string {
	return fmt.Sprintf(`Command Usage:
  cp `)
}

func (c *cpCommand) Init(env *env.Environment) (err error) {
	c.env = env
	c.cli, _ = client.NewStorageClient(env)
	opts := flag.NewFlagSet("cp",flag.ExitOnError)
	opts.BoolVar(&c.recursive, "r", false, "recursively upload")
	opts.Usage = func() {
		fmt.Fprintln(os.Stdout, c.Usage())
	}
	c.opts = opts
	return
}

func (c *cpCommand) Run(args []string) (err error) {
	var (
		sourceBucket = ""
		sourceKey    = ""
		distBucket	 = ""
		distKey		 = ""
	)
	if len(args) == 0 {
		return ErrArgument
	}
	c.opts.Parse(args)
	argv := c.opts.Args()
	if len(argv) < 2 {
		return ErrArgument
	}
	source := strings.Split(argv[0], ":")
	sourceBucket = source[0]
	sourceKey = source[1]
	dist := strings.Split(argv[1], ":")
	distBucket = dist[0]
	distKey = dist[1]
	// object -> bucket
	if !strings.HasSuffix(sourceKey, "/") && distKey == ""{
		slice := strings.Split(sourceKey, "/")
		err = c.cli.PutObjectCopy(sourceBucket,sourceKey,distBucket,slice[len(slice)-1])
		if err != nil {
			return
		}
		if c.env.Verbose {
			fmt.Fprintf(os.Stdout, "copy: %s:%s -> %s:%s\n", sourceBucket, sourceKey, distBucket, slice[len(slice)-1])
		}
		return nil
	}

	// object -> dir
	if !strings.HasSuffix(sourceKey, "/") && distKey != "" {
		slice := strings.Split(sourceKey, "/")
		if !strings.HasSuffix(distKey, "/") {
			distKey += "/"
		}
		targetKey := distKey + slice[len(slice)-1]
		err = c.cli.PutObjectCopy(sourceBucket,sourceKey,distBucket,targetKey)
		if err != nil {
			return
		}
		if c.env.Verbose {
			fmt.Fprintf(os.Stdout, "copy: %s:%s -> %s:%s\n", sourceBucket, sourceKey, distBucket, targetKey)
		}
		return nil
	}

	if c.recursive {
		// dir -> bucket
		if distKey == "" {
			if !strings.HasSuffix(sourceKey, "/") {
				sourceKey += "/"
			}
			listing, err := c.cli.ListObjects(sourceBucket, sourceKey, "", "/", 1000)
			if err != nil {
				return err
			}
			for _, n := range listing.Summaries {
				err = c.cli.PutObjectCopy(sourceBucket, n.Key, distBucket, n.Key)
				if err != nil {
					return err
				}
				if c.env.Verbose {
					fmt.Fprintf(os.Stdout, "copy: %s:%s -> %s:%s\n", sourceBucket, n.Key, distBucket, n.Key)
				}
			}
			return nil
		}

		// dir -> dir
		if distKey != "" {
			if !strings.HasSuffix(sourceKey, "/") {
				sourceKey += "/"
			}
			listing, err := c.cli.ListObjects(sourceBucket, sourceKey, "", "/", 1000)
			if err != nil {
				return err
			}
			if !strings.HasSuffix(distKey, "/") {
				distKey += "/"
			}
			var targetKey string
			var slice 	  []string
			for _, n := range listing.Summaries {
				slice = strings.Split(n.Key, "/")
				targetKey = distKey + slice[len(slice)-2] + "/" + slice[len(slice)-1]
				err = c.cli.PutObjectCopy(sourceBucket, n.Key, distBucket, targetKey)
				if c.env.Verbose {
					fmt.Fprintf(os.Stdout, "copy: %s:%s -> %s:%s\n", sourceBucket, n.Key, distBucket, targetKey)
				}
				if err != nil {
					return err
				}
			}
			return nil
		}
	} else {
		return fmt.Errorf("%q:%q is a directory (not copy)", sourceBucket, sourceKey)
	}
	return
}

func init() {
	Commands.Register(new(cpCommand), "cp")
}

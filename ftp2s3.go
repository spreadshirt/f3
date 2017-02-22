package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"git.spreadomat.net/sprd/ftp2s3/ftplib"
	ftp "github.com/goftp/server"
	"gopkg.in/urfave/cli.v1"
)

// AppName is the name of the program.
const AppName string = "ftp2s3"

// Version is the current version of ftps3.
var Version string

func main() {
	app := cli.NewApp()
	app.Name = AppName
	app.Usage = "an FTP to s3/ceph bridge"
	app.Version = Version
	app.Description = "A tool that acts as a bridge between FTP and a s3/ceph bucket"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "ftp-addr",
			Value: "127.0.0.1:21",
			Usage: "Address of the FTP server interface, default: 127.0.0.1:21",
		},
		cli.StringFlag{
			Name:  "ftp-root",
			Usage: "Root path of the FTP server, default is the current working directory",
		},
		cli.StringFlag{
			Name:  "ftp-features",
			Value: "ls",
			Usage: "FTP feature set, default is empty. Example: --ftp-features=\"get,put,ls\"",
		},
		cli.BoolFlag{
			Name:  "ftp-no-overwrite",
			Usage: "Prevent files from being overwritten",
		},
		cli.StringFlag{
			Name:  "s3-credentials",
			Usage: "AccessKey:SecretKey",
		},
		cli.StringFlag{
			Name:  "s3-bucket",
			Usage: "URL of the s3 bucket, e.g. https://some-bucket.s3.amazonaws.com",
		},
		cli.StringFlag{
			Name:  "s3-region",
			Value: "default",
			Usage: "Region where the s3 bucket is located in",
		},
	}
	app.Action = run
	app.Run(os.Args)
}

func run(context *cli.Context) error {
	if context.NArg() < 1 {
		return fmt.Errorf("not enough arguments, path to FTP credentials file is missing")
	}

	creds, err := ftplib.AuthenticatorFromFile(context.Args().First())
	if err != nil {
		return err
	}

	ftpHost, ftpPort, err := splitFtpAddr(context.String("ftp-addr"))
	if err != nil {
		return err
	}

	factory, err := ftplib.NewDriverFactory(&ftplib.FactoryConfig{
		FtpRoot:        context.String("ftp-root"),
		FtpFeatures:    context.String("ftp-features"),
		FtpNoOverwrite: context.Bool("ftp-no-overwrite"),
		S3Credentials:  context.String("s3-credentials"),
		S3BucketURL:    context.String("s3-bucket"),
		S3Region:       context.String("s3-region"),
	})
	if err != nil {
		return err
	}

	ftpServer := ftp.NewServer(&ftp.ServerOpts{
		Factory:        factory,
		Auth:           creds,
		Name:           AppName,
		Hostname:       ftpHost,
		Port:           ftpPort,
		WelcomeMessage: fmt.Sprintf("%s says hello!", AppName),
	})
	return ftpServer.ListenAndServe()
}

func splitFtpAddr(addr string) (string, int, error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return "", -1, fmt.Errorf("Empty FTP address")
	}
	parts := strings.SplitN(addr, ":", 2)
	host := parts[0]
	port := uint64(21)
	if len(parts) < 2 { // no port given
		return host, int(port), nil
	}

	port, err := strconv.ParseUint(parts[1], 10, 16)
	if err != nil {
		return host, -1, fmt.Errorf("Invalid FTP port %q: %s", parts[1], err)
	}

	return host, int(port), err
}

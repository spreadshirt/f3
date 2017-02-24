package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"git.spreadomat.net/sprd/ftp2s3/server"
	ftp "github.com/klingtnet/goftp"
	"github.com/sirupsen/logrus"
	cli "gopkg.in/urfave/cli.v1"
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
			Name:  "features",
			Value: server.DefaultFeatureSet,
			Usage: "Feature set, default is empty. Example: --features=\"get,put,ls\"",
		},
		cli.BoolFlag{
			Name:  "no-overwrite",
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
			Value: server.DefaultRegion,
			Usage: "Region where the s3 bucket is located in",
		},
		cli.BoolFlag{
			Name:  "verbose",
			Usage: "Print what is being done",
		},
	}
	app.Action = run
	err := app.Run(os.Args)
	if err == nil {
		logrus.WithFields(logrus.Fields{"msg": err}).Fatal(err)
	}
}

func run(context *cli.Context) error {
	if context.NArg() < 1 {
		return fmt.Errorf("not enough arguments, path to FTP credentials file is missing")
	}

	if context.Bool("verbose") {
		logrus.SetLevel(logrus.DebugLevel)
	}

	credentialsFilename := context.Args().First()
	logrus.Debugf("Trying to read credentials file: %q", credentialsFilename)
	creds, err := server.AuthenticatorFromFile(credentialsFilename)
	if err != nil {
		return err
	}

	ftpHost, ftpPort, err := splitFtpAddr(context.String("ftp-addr"))
	if err != nil {
		return err
	}

	factory, err := server.NewDriverFactory(&server.FactoryConfig{
		FtpFeatures:    context.String("features"),
		FtpNoOverwrite: context.Bool("no-overwrite"),
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
		Logger:         &server.FTPLogger{},
	})
	logrus.Infof("FTP server starts listening on \"%s:%d\"", ftpHost, ftpPort)
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

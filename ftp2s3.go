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
	}
	app.Action = run
	app.Run(os.Args)
}

func run(context *cli.Context) error {
	if context.NArg() < 1 {
		return fmt.Errorf("Not enough arguments, path to credentials file is missing!")
	}

	creds, err := ftplib.AuthenticatorFromFile(context.Args().First())
	if err != nil {
		return err
	}

	ftpRoot := context.String("ftp-root")
	// set FTP root to the current working directory if unset
	if ftpRoot == "" {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		ftpRoot = wd
	}

	ftpAddr := context.String("ftp-addr")
	if ftpAddr == "" {
		return fmt.Errorf("FTP address is empty")
	}
	parts := strings.SplitN(ftpAddr, ":", 2)
	ftpHost := "127.0.0.1"
	ftpPort := int64(21)
	if len(parts) == 1 {
		ftpHost = parts[0]
	} else if len(parts) > 1 {
		ftpHost = parts[0]
		ftpPort, err = strconv.ParseInt(parts[1], 10, 16)
		if err != nil {
			return err
		}
	}

	factory, err := ftplib.NewDriverFactory(ftpRoot, context.String("ftp-features"), context.Bool("ftp-no-overwrite"))
	if err != nil {
		return err
	}
	opts := ftp.ServerOpts{
		Factory:        factory,
		Auth:           creds,
		Name:           AppName,
		Hostname:       ftpHost,
		Port:           int(ftpPort),
		WelcomeMessage: fmt.Sprintf("%s says hello!", AppName),
	}
	ftpServer := ftp.NewServer(&opts)
	return ftpServer.ListenAndServe()
}

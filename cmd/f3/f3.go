package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spreadshirt/f3/meta"
	"github.com/spreadshirt/f3/server"

	ftp "github.com/goftp/server"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// AppName is the name of the program.
const AppName string = "f3"

type cliFlags struct {
	ftpAddr           string
	features          string
	noOverwrite       bool
	s3Credentials     string
	s3Bucket          string
	s3Region          string
	disableCloudwatch bool
	verbose           bool
}

func main() {
	flags := cliFlags{}

	cmd := &cobra.Command{
		Use:   fmt.Sprintf("%s /path/to/ftp-credentials.txt", os.Args[0]),
		Short: "f3 acts like a bridge between FTP and an s3 bucket",
		Long: `f3 is a bridge between FTP and an s3 bucket.
It maps FTP commands to s3 equivalents and stores uploaded files as objects in an s3 bucket.
The feature set of the FTP server can be set very fine grained, e.g. you can only allow 'ls' and 'get' operations.
Additionally, you can prevent objects from getting overwritten.

See https://github.com/spreadshirt/f3 for details.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 1 {
				cmd.Usage()
				return
			}
			if args[0] == "version" {
				fmt.Printf("%s %s\n", AppName, meta.Version)
				return
			}
			err := run(args[0], flags)
			if err != nil {
				logrus.WithFields(logrus.Fields{"msg": err}).Fatal(err)
			}
		},
	}

	cmd.PersistentFlags().StringVar(&flags.ftpAddr, "ftp-addr", "127.0.0.1:21", "Address of the FTP server interface, default: 127.0.0.1:21")
	cmd.PersistentFlags().StringVar(&flags.features, "features", server.DefaultFeatureSet, fmt.Sprintf("Feature set, default is empty. Default: --features=%q", server.DefaultFeatureSet))
	cmd.PersistentFlags().BoolVar(&flags.noOverwrite, "no-overwrite", false, "Prevent files from being overwritten")
	cmd.PersistentFlags().StringVar(&flags.s3Credentials, "s3-credentials", "", "AccessKey:SecretKey")
	cmd.PersistentFlags().StringVar(&flags.s3Bucket, "s3-bucket", "", "URL of the s3 bucket, e.g. https://some-bucket.s3.amazonaws.com")
	cmd.PersistentFlags().StringVar(&flags.s3Region, "s3-region", server.DefaultRegion, "Region where the s3 bucket is located in")
	cmd.PersistentFlags().BoolVar(&flags.disableCloudwatch, "disable-cloudwatch", false, "Disable CloudWatch metrics")
	cmd.PersistentFlags().BoolVarP(&flags.verbose, "verbose", "v", false, "Print what is being done")

	err := cmd.Execute()
	if err != nil {
		logrus.WithFields(logrus.Fields{"msg": err}).Fatal(err)
	}
}

func run(credentialsFilename string, flags cliFlags) error {
	if flags.verbose {
		logrus.SetLevel(logrus.DebugLevel)
	}

	logrus.Debugf("Trying to read credentials file: %q", credentialsFilename)
	creds, err := server.AuthenticatorFromFile(credentialsFilename)
	if err != nil {
		return errors.Wrapf(err, "Failed to read credentials file %q", credentialsFilename)
	}

	ftpAddr := getEnvOrDefault("FTP_ADDR", flags.ftpAddr)
	ftpHost, ftpPort, err := splitFtpAddr(ftpAddr)
	if err != nil {
		return errors.Wrapf(err, "Failed to split %q in host and port", ftpAddr)
	}

	factory, err := server.NewDriverFactory(&server.FactoryConfig{
		FtpFeatures:       getEnvOrDefault("FTP_FEATURES", flags.features),
		FtpNoOverwrite:    flags.noOverwrite,
		S3Credentials:     getEnvOrDefault("S3_CREDENTIALS", flags.s3Credentials),
		S3BucketURL:       getEnvOrDefault("BUCKET_URL", flags.s3Bucket),
		S3Region:          getEnvOrDefault("S3_REGION", flags.s3Region),
		DisableCloudWatch: flags.disableCloudwatch,
	})
	if err != nil {
		return errors.Wrapf(err, "Failed to instantiate new driver factory")
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

	return host, int(port), nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

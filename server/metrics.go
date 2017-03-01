package server

import (
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
)

type MetricsSender interface {
	SendPut(size int64, timestamp time.Time) error
	SendGet(size int64, timestamp time.Time) error
}

type CloudwatchSender struct {
	metrics  cloudwatchiface.CloudWatchAPI
	hostname string
}

func NewCloudwatchSender(awsSession *session.Session) (MetricsSender, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	return &CloudwatchSender{
		hostname: hostname,
		metrics:  cloudwatch.New(awsSession),
	}, nil
}

func (c *CloudwatchSender) SendPut(size int64, timestamp time.Time) error {
	_, err := c.metrics.PutMetricData(&cloudwatch.PutMetricDataInput{
		Namespace: aws.String("f3"),
		MetricData: []*cloudwatch.MetricDatum{
			&cloudwatch.MetricDatum{
				MetricName: aws.String("GET"),
				Timestamp:  &timestamp,
				Unit:       aws.String("Bytes"),
				Value:      aws.Float64(float64(size)),
				Dimensions: []*cloudwatch.Dimension{&cloudwatch.Dimension{
					Name:  aws.String("Hostname"),
					Value: aws.String(c.hostname),
				}},
			},
		},
	})
	return err
}

func (c *CloudwatchSender) SendGet(size int64, timestamp time.Time) error {
	_, err := c.metrics.PutMetricData(&cloudwatch.PutMetricDataInput{
		Namespace: aws.String("f3"),
		MetricData: []*cloudwatch.MetricDatum{
			&cloudwatch.MetricDatum{
				MetricName: aws.String("PUT"),
				Timestamp:  &timestamp,
				Unit:       aws.String("Bytes"),
				Value:      aws.Float64(float64(size)),
				Dimensions: []*cloudwatch.Dimension{&cloudwatch.Dimension{
					Name:  aws.String("Hostname"),
					Value: aws.String(c.hostname),
				}},
			},
		},
	})
	return err
}

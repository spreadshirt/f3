package server

import (
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"github.com/pkg/errors"
)

// MetricsSender defines methods for sending data to a metrics provider.
type MetricsSender interface {
	// SendPut sends the size of a stored (PUT) object and the operation's timestamp.
	SendPut(size int64, timestamp time.Time) error
	// SendGet sends the size of a served (GET) object and the operation's timestamp.
	SendGet(size int64, timestamp time.Time) error
}

// NopSender returns immediately.
type NopSender struct{}

// SendPut returns nil.
func (n NopSender) SendPut(size int64, timestamp time.Time) error { return nil }

// SendGet returns nil.
func (n NopSender) SendGet(size int64, timestamp time.Time) error { return nil }

// CloudwatchSender implements MetricsSender for amazon's cloudwatch service.
type CloudwatchSender struct {
	metrics  cloudwatchiface.CloudWatchAPI
	hostname string
}

// NewCloudwatchSender returns a new CloudwatchSender for the given session.
func NewCloudwatchSender(awsSession *session.Session) (MetricsSender, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get hostname")
	}
	return &CloudwatchSender{
		hostname: hostname,
		metrics:  cloudwatch.New(awsSession),
	}, nil
}

// SendPut stores the metric data for a PUT operation in cloudwatch.
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
	if err != nil {
		logAwsError(intoAwsError(err))
		return errors.Wrapf(err, "Failed to send cloudwatch PUT metric")
	}
	return nil
}

// SendGet stores the metric data for a GET operation in cloudwatch.
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
	if err != nil {
		logAwsError(intoAwsError(err))
		return errors.Wrapf(err, "Failed to send cloudwatch GET metric")
	}
	return nil
}

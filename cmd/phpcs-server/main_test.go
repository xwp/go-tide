package main

import (
	"bytes"
	"log"
	"os"
	"testing"
	"time"

	"github.com/wptide/pkg/env"
	"github.com/wptide/pkg/message"
	"github.com/wptide/pkg/payload"
)

var (
	envTest = map[string]string{
		// Tide API config
		"TIDE_API_AUTH_URL": "http://tide.local/api/tide/v1/auth",
		"TIDE_API_HOST":     "tide.local",
		"TIDE_API_PROTOCOL": "http",
		"TIDE_API_KEY":      "tideapikey",
		"TIDE_API_SECRET":   "tideapisecret",
		"TIDE_API_VERSION":  "v1",
		// AWS SQS settings
		"PHPCS_SQS_VERSION":    "2012-11-05",
		"PHPCS_SQS_REGION":     "us-west-2",
		"PHPCS_SQS_KEY":        "sqskey",
		"PHPCS_SQS_SECRET":     "sqssecret",
		"PHPCS_SQS_QUEUE_NAME": "test-queue",
		//
		// AWS S3 settings
		"PHPCS_S3_REGION":      "us-west-2",
		"PHPCS_S3_KEY":         "s3key",
		"PHPCS_S3_SECRET":      "s3secret",
		"PHPCS_S3_BUCKET_NAME": "test-bucket",
		//
		// PHPCS Server settings
		"PHPCS_CONCURRENT_AUDITS":      "1",
	}
)

func Test_initProcesses(t *testing.T) {

	type args struct {
		source <-chan message.Message
		config processConfig
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"Valid Processes",
			args{
				make(chan message.Message),
				processConfig{
					igTempFolder:      "./testdata/tmp",
					phpcsTempFolder:      "./testdata/tmp",
					phpcsStorageProvider: &mockStorage{},
					resPayloaders: map[string]payload.Payloader{
						"tide": &mockPayloader{},
					},
				},
			},
			false,
		},
		{
			"No Config",
			args{
				source: make(chan message.Message),
			},
			true,
		},
		{
			"No Source",
			args{
				config: processConfig{
					igTempFolder:      "./testdata/tmp",
					phpcsTempFolder:      "./testdata/tmp",
					phpcsStorageProvider: &mockStorage{},
					resPayloaders: map[string]payload.Payloader{
						"tide": &mockPayloader{},
					},
				},
			},
			true,
		},
		{
			"Ingest temp folder missing",
			args{
				make(chan message.Message),
				processConfig{
					phpcsTempFolder:      "./testdata/tmp",
					phpcsStorageProvider: &mockStorage{},
					resPayloaders: map[string]payload.Payloader{
						"tide": &mockPayloader{},
					},
				},
			},
			true,
		},
		{
			"Lighthouse temp folder missing",
			args{
				make(chan message.Message),
				processConfig{
					igTempFolder:      "./testdata/tmp",
					phpcsStorageProvider: &mockStorage{},
					resPayloaders: map[string]payload.Payloader{
						"tide": &mockPayloader{},
					},
				},
			},
			true,
		},
		{
			"Lighthouse storage provider missing",
			args{
				make(chan message.Message),
				processConfig{
					igTempFolder: "./testdata/tmp",
					phpcsTempFolder: "./testdata/tmp",
					resPayloaders: map[string]payload.Payloader{
						"tide": &mockPayloader{},
					},
				},
			},
			true,
		},
		{
			"Response payloaders missing",
			args{
				make(chan message.Message),
				processConfig{
					igTempFolder:      "./testdata/tmp",
					phpcsTempFolder:      "./testdata/tmp",
					phpcsStorageProvider: &mockStorage{},
				},
			},
			true,
		},
		{
			"Valid Processes with messages",
			args{
				make(chan message.Message),
				processConfig{
					igTempFolder:      "./testdata/tmp",
					phpcsTempFolder:      "./testdata/tmp",
					phpcsStorageProvider: &mockStorage{},
					resPayloaders: map[string]payload.Payloader{
						"tide": &mockPayloader{},
					},
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			_, err := initProcesses(tt.args.source, tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("initProcesses() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_main(t *testing.T) {

	b := bytes.Buffer{}
	log.SetOutput(&b)

	var value string
	for key, val := range envTest {

		// Key is set, so retain the value when the test is finished.
		if value = os.Getenv(key); value != "" {
			os.Unsetenv(key)
			defer func() { os.Setenv(key, value) }()
		}

		// Set the test value.
		os.Setenv(key, val)
	}

	setupConfig()

	// Use the mockTide for Tide
	currentClient := TideClient
	TideClient = &mockTide{}
	defer func() { TideClient = currentClient }()

	cMessageProvider := messageProvider
	messageProvider = &mockMessageProvider{}
	defer func() { messageProvider = cMessageProvider }()

	type args struct {
		messageChannel chan message.Message
		timeOut        time.Duration
		msg            message.Message
		parseFlags     bool
		version        bool
		authError      bool
		flagUrl        *string
		flagOutput     *string
		flagVisibility *string
		altConfig      *processConfig
	}

	tests := []struct {
		name     string
		args     args
	}{
		{
			"Run Main - Process Config Error (missing)",
			args{
				messageChannel: make(chan message.Message, 1),
				timeOut:        1,
				msg:            message.Message{},
				altConfig:      &processConfig{},
			},
		},
		{
			"Run Main",
			args{
				timeOut: 1,
				version: true,
			},
		},
		{
			"Run Main - Custom Message",
			args{
				messageChannel: make(chan message.Message, 1),
				timeOut:        1,
				msg:            message.Message{},
			},
		},
		{
			"Run Main - Version flag set",
			args{
				timeOut:    1,
				parseFlags: true,
			},
		},
		{
			"Run Main - Auth Error",
			args{
				timeOut:   1,
				authError: true,
			},
		},
		{
			"Run Main - Output Flag set",
			args{
				timeOut:    1,
				flagOutput: &[]string{"./testdata/report.json"}[0],
			},
		},
		{
			"Run Main - URL and Visibility Flag set",
			args{
				timeOut:        1,
				flagUrl:        &[]string{testFileServer.URL + "/test.zip"}[0],
				flagVisibility: &[]string{"public"}[0],
			},
		},
		{
			"Run Main - Process Error",
			args{
				timeOut: 0,
				version: true,
				// Invalid config. Will cause a process.Run() error.
				altConfig: &processConfig{
					igTempFolder:      "./testdata/tmp",
					phpcsTempFolder:      "./testdata/tmp",
					phpcsStorageProvider: &mockStorage{},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// Parse the flags.
			bParseFlags = tt.args.parseFlags

			// Alternate config.
			if tt.args.altConfig != nil {
				oldConf := procCfg
				procCfg = *tt.args.altConfig
				defer func() {
					procCfg = oldConf
				}()
			}

			// -output
			if tt.args.flagOutput != nil && *tt.args.flagOutput != "" {
				flagOutput = tt.args.flagOutput
			}

			// -url
			if tt.args.flagUrl != nil && *tt.args.flagUrl != "" {
				flagUrl = tt.args.flagUrl
			}

			// -visibility
			if tt.args.flagVisibility != nil && *tt.args.flagVisibility != "" {
				flagVisibility = tt.args.flagVisibility
			}

			if tt.args.version {
				bVersion = &[]bool{true}[0]
				Version = "0.0.1-test"
				Build = "12345"
			}

			if tt.args.authError {
				oldId := tideConfig.id
				tideConfig.id = "error"
				defer func() {
					tideConfig.id = oldId
				}()
			}

			// Run as goroutine and wait for terminate signal.
			go main()

			if tt.args.messageChannel != nil {
				oldCMessage := cMessage
				cMessage = tt.args.messageChannel
				cMessage <- tt.args.msg
				defer func() {
					cMessage = oldCMessage
				}()
			}

			// Sleep for one second. Allows for one poll action.
			time.Sleep(time.Millisecond * 100 * tt.args.timeOut)
			terminateChannel <- struct{}{}
		})
	}
}

func setupConfig() {
	// Setup queueConfig
	queueConfig = struct {
		region string
		key    string
		secret string
		queue  string
	}{
		env.GetEnv("PHPCS_SQS_REGION", ""),
		env.GetEnv("PHPCS_SQS_KEY", ""),
		env.GetEnv("PHPCS_SQS_SECRET", ""),
		env.GetEnv("PHPCS_SQS_QUEUE_NAME", ""),
	}

	tideConfig = struct {
		id           string
		secret       string
		authEndpoint string
		host         string
		protocol     string
		version      string
	}{
		env.GetEnv("TIDE_API_KEY", ""),
		env.GetEnv("TIDE_API_SECRET", ""),
		env.GetEnv("TIDE_API_AUTH_URL", ""),
		env.GetEnv("TIDE_API_HOST", ""),
		env.GetEnv("TIDE_API_PROTOCOL", ""),
		env.GetEnv("TIDE_API_VERSION", ""),
	}

	s3Config = struct {
		region string
		key    string
		secret string
		bucket string
	}{
		env.GetEnv("PHPCS_S3_REGION", ""),
		env.GetEnv("PHPCS_S3_KEY", ""),
		env.GetEnv("PHPCS_S3_SECRET", ""),
		env.GetEnv("PHPCS_S3_BUCKET_NAME", ""),
	}
}

func Test_pollProvider(t *testing.T) {
	type args struct {
		c        chan message.Message
		provider message.MessageProvider
		buffer   chan struct{}
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"Poll Messages",
			args{
				make(chan message.Message),
				&mockMessageProvider{
					"message",
				},
				make(chan struct{}, 1),
			},
		},
		{
			"Poll Messages - Critical Error",
			args{
				make(chan message.Message),
				&mockMessageProvider{
					"critical",
				},
				make(chan struct{}, 1),
			},
		},
		{
			"Poll Messages - Quota Error",
			args{
				make(chan message.Message),
				&mockMessageProvider{
					"quota",
				},
				make(chan struct{}, 1),
			},
		},
		{
			"Poll Messages - Message Length Error",
			args{
				make(chan message.Message),
				&mockMessageProvider{
					"lenError",
				},
				make(chan struct{}, 1),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pollProvider(tt.args.c, tt.args.provider, tt.args.buffer)
		})
	}
}

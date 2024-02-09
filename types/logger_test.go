package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ibc-scouts/ibc-interceptor/types"
)

func mockLogHandler(array *[]string, logLevel string) log.Handler {
	baseHandler := log.FuncHandler(func(r *log.Record) error {
		*array = append(*array, r.Msg)
		return nil
	})

	lvl, err := log.LvlFromString(logLevel)
	if err != nil {
		panic(err)
	}

	return log.LvlFilterHandler(lvl, baseHandler)
}

func TestLogger(t *testing.T) {
	testCases := []struct {
		name     string
		logLevel string
		expPass  bool
	}{
		{
			"success: log level trace",
			"trace",
			true,
		},
		{
			"success: log level debug",
			"debug",
			true,
		},
		{
			"success: log level info",
			"info",
			true,
		},
		{
			"success: log level warn",
			"warn",
			true,
		},
		{
			"success: log level error",
			"error",
			true,
		},
		{
			"success: log level crit",
			"crit",
			true,
		},
		{
			"failure: log level unknown",
			"unknown",
			false,
		},
		{
			"failure: log level empty",
			"",
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			logger, err := types.NewCompositeLogger(tc.logLevel)

			if tc.expPass {
				require.NoError(t, err)
				require.NotNil(t, logger)

				var logArray []string
				logger.SetHandler(mockLogHandler(&logArray, tc.logLevel))

				logger.Trace("trace")
				logger.Debug("debug")
				logger.Info("info")
				logger.Warn("warn")
				logger.Error("error")
				// crit kills the application

				switch tc.logLevel {
				case "trace":
					require.Len(t, logArray, 5)
				case "debug":
					require.Len(t, logArray, 4)
					require.NotContains(t, logArray, "trace")
				case "info":
					require.Len(t, logArray, 3)
					require.NotContains(t, logArray, "trace")
					require.NotContains(t, logArray, "debug")
				case "warn":
					require.Len(t, logArray, 2)
					require.Contains(t, logArray, "warn")
					require.Contains(t, logArray, "error")
				case "error":
					require.Len(t, logArray, 1)
					require.Contains(t, logArray, "error")
				case "crit":
					require.Len(t, logArray, 0)
				default:
					t.FailNow()
				}
			} else {
				require.Error(t, err)
				require.Nil(t, logger)
			}
		})
	}
}

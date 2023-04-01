package logger

import (
	"testing"

	"github.com/stretchr/testify/require"
	"tideland.dev/go/audit/capture"
)

var levels = map[string]int{debugLvl: 0, errorLvl: 3}

func TestLogger(t *testing.T) {
	testCases := []struct {
		name         string
		loggerLvl    string
		msgLvl       string
		msg          string
		expLogFields []string
	}{
		{
			name:      "loggerLvl:error-msgLvl:debug",
			loggerLvl: "error",
			msgLvl:    "debug",
			msg:       "any debug msg",
		},
		{
			name:      "loggerLvl:error-msgLvl:error",
			loggerLvl: "error",
			msgLvl:    "error",
			msg:       "any err msg",
			expLogFields: []string{
				`"level":"error"`,
				`"time":`,
				`"message":"any err msg"`,
				`"caller":`,
			},
		},
		{
			name:      "loggerLvl:debug-msgLvl:debug",
			loggerLvl: "debug",
			msgLvl:    "debug",
			msg:       "any debug msg",
			expLogFields: []string{
				`"level":"debug"`,
				`"time":`,
				`"message":"any debug msg"`,
				`"caller":`,
			},
		},
		{
			name:      "loggerLvl:debug-msgLvl:error",
			loggerLvl: "debug",
			msgLvl:    "error",
			msg:       "any err msg",
			expLogFields: []string{
				`"level":"error"`,
				`"time":`,
				`"message":"any err msg"`,
				`"caller":`,
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var loggerMethod func(Logger, string, ...any)
			switch tc.msgLvl {
			case debugLvl:
				loggerMethod = Logger.Debug
			case errorLvl:
				loggerMethod = Logger.Error
			}

			capturedLogOut := capture.Stdout(func() {
				logger := NewLogger(tc.loggerLvl)
				loggerMethod(logger, tc.msg)
			})

			if levels[tc.msgLvl] < levels[tc.loggerLvl] {
				require.Equal(t, "", capturedLogOut.String(),
					"unexpected log, msgLvl (%s) < loggerLvl (%s)", tc.msgLvl, tc.loggerLvl)
				return
			}

			for _, expLogField := range tc.expLogFields {
				require.Contains(t, capturedLogOut.String(), expLogField)
			}
		})
	}
}

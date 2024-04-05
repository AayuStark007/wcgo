package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func path(fileName string) string {
	curDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	parent, err := filepath.Abs(filepath.Dir(curDir))
	if err != nil {
		panic(err)
	}

	return parent + "/" + fileName
}

func TestWCContext_String(t *testing.T) {
	type fields struct {
		flagBytes bool
		flagLines bool
		flagWords bool
		flagChars bool
		files     []string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			"text_bytes",
			fields{
				true, false, false, false, []string{path("examples/test.txt")},
			},
			fmt.Sprintf("%d %s\n", 342190, path("examples/test.txt")),
		},
		{
			"text_lines",
			fields{
				false, true, false, false, []string{path("examples/test.txt")},
			},
			fmt.Sprintf("%d %s\n", 7145, path("examples/test.txt")),
		},
		{
			"text_words",
			fields{
				false, false, true, false, []string{path("examples/test.txt")},
			},
			fmt.Sprintf("%d %s\n", 58164, path("examples/test.txt")),
		},
		{
			"text_chars",
			fields{
				false, false, false, true, []string{path("examples/test.txt")},
			},
			fmt.Sprintf("%d %s\n", 339292, path("examples/test.txt")),
		},
		{
			"text_all",
			fields{
				false, false, false, false, []string{path("examples/test.txt")},
			},
			fmt.Sprintf("%d %d %d %s\n", 7145, 58164, 342190, path("examples/test.txt")),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, err := New(tt.fields.files, tt.fields.flagBytes, tt.fields.flagLines, tt.fields.flagWords, tt.fields.flagChars)
			if err != nil {
				t.Errorf("New() error = %v, wantErr false", err)
				return
			}
			ctx.Compute()
			if got := ctx.String(); got != tt.want {
				t.Errorf("WCContext.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

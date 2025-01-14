package cli

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestNewFederationtCmd(t *testing.T) {
	expected := &cobra.Command{
		Use:   "federation",
		Short: "Manage federation relationships",
		Long:  "Run this command to approve and deny relationships",
	}
	assert.ObjectsAreEqual(expected, NewFederationtCmd())
}

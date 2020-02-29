package cmd

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_showConfigCmd(t *testing.T) {
	fmt.Println("test")
	assert.NotPanics(t, func() { showConfigCmd.Run(nil, nil) })
}

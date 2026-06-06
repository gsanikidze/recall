package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func promptYN(question string) (bool, error) {
	fmt.Printf("%s [y/N]: ", question)

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	in := strings.TrimSpace(line)
	return in == "y" || in == "Y", nil
}

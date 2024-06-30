package config

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Configuration struct {
	SplitFlap SplitFlapConfig
}

type SplitFlapConfig struct {
	ModuleCount        int
	DriverCount        int
	AlphabetOffset     []string
	AlphabetESP32Order string
}

var Config *Configuration

func init() {
	err := godotenv.Load(".env")
	if err != nil {
		panic("Error loading .env file")
		return
	}

	moduleCount, err := strconv.Atoi(GetVar("SPLITFLAP_MODULE_COUNT"))
	if err != nil {
		panic("SPLITFLAP_MODULE_COUNT defined in .env must be a number")
	}

	alphabetOffset := strings.Split(GetVar("ALPHABET_OFFSET"), ",")
	if len(Config.SplitFlap.AlphabetOffset) != moduleCount {
		panic("ALPHABET_OFFSET does not contain enough entries, must match SPLITFLAP_MODULE_COUNT")
	}

	Config = &Configuration{
		SplitFlap: SplitFlapConfig{
			ModuleCount:        moduleCount,
			DriverCount:        int(math.Round(float64(moduleCount)/6.0 + .5)),
			AlphabetOffset:     alphabetOffset,
			AlphabetESP32Order: GetVar("ALPHABET_ORDER"),
		},
	}
}

func GetVar(str string) string {
	v := os.Getenv(str)
	if v == "" {
		panic(fmt.Sprintf("missing env var: %s", str))
	}

	return v
}

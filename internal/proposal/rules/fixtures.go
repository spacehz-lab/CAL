package rules

import (
	"encoding/base64"

	"github.com/spacehz-lab/cal/internal/proposal"
)

const testPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAUAAAAFCAIAAAACDbGyAAAAFElEQVR4nGM8ceIEA27AhEduBEsDABUMAYuJ1HWoAAAAAElFTkSuQmCC"

func probeFixtures(capabilityID string) []proposal.Fixture {
	switch capabilityID {
	case capabilityDocumentConvert:
		return []proposal.Fixture{{
			Input:    inputSource,
			Filename: "input.txt",
			Content:  "cal probe input\n",
		}}
	case capabilityImageResize:
		return []proposal.Fixture{{
			Input:    inputSource,
			Filename: "input.png",
			Content:  mustDecodeString(testPNGBase64),
		}}
	default:
		return nil
	}
}

func mustDecodeString(value string) string {
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		panic(err)
	}
	return string(decoded)
}

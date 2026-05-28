package actx

import (
	"github.com/yunerou/niarb/shared/encdec"
)

func (a *aContext) SetEncoderDecoder(e encdec.Encoder, d encdec.Decoder) {
	a.data.encoder = e
	a.data.decoder = d
}

func (a *aContext) GetEncoder() encdec.Encoder {
	if a.data.encoder == nil {
		panic("encoder is not set in aContext")
	}
	return a.data.encoder
}
func (a *aContext) GetDecoder() encdec.Decoder {
	if a.data.decoder == nil {
		panic("decoder is not set in aContext")
	}
	return a.data.decoder
}

/*
 *
 */

package qperrors

const (
	PARTS_ERROR_UNKNOWN_PACKET_TYPE = 1000
	PARTS_ERROR_HANDSHAKE_INVALID   = 1001
	PARTS_ERROR_HANDSHAKE_TIMEOUT   = 1002
)

var PartsError map[int]string

func init() {
	PartsError = make(map[int]string)
	PartsError[PARTS_ERROR_UNKNOWN_PACKET_TYPE] = "Unknown packet type"
	PartsError[PARTS_ERROR_HANDSHAKE_INVALID] = "Handshake invalid"
	PartsError[PARTS_ERROR_HANDSHAKE_TIMEOUT] = "Handshake timeout"
}

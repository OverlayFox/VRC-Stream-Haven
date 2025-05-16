package srt

import (
	"fmt"

	srt "github.com/datarhei/gosrt"
)

// rejectionReasonToString is a helper function to convert RejectionReason to a readable string
func rejectionReasonToString(reason srt.RejectionReason) string {
	switch reason {
	case srt.REJ_UNKNOWN:
		return "unknown"
	case srt.REJ_SYSTEM:
		return "system function error"
	case srt.REJ_PEER:
		return "rejected by peer"
	case srt.REJ_RESOURCE:
		return "resource allocation problem"
	case srt.REJ_ROGUE:
		return "incorrect data in handshake"
	case srt.REJ_BACKLOG:
		return "listener's backlog exceeded"
	case srt.REJ_IPE:
		return "internal program error"
	case srt.REJ_CLOSE:
		return "socket is closing"
	case srt.REJ_VERSION:
		return "peer is older version than server's min"
	case srt.REJ_RDVCOOKIE:
		return "rendezvous cookie collision"
	case srt.REJ_BADSECRET:
		return "wrong password"
	case srt.REJ_UNSECURE:
		return "password required or unexpected"
	case srt.REJ_MESSAGEAPI:
		return "stream flag collision"
	case srt.REJ_CONGESTION:
		return "incompatible congestion-controller type"
	case srt.REJ_FILTER:
		return "incompatible packet filter"
	case srt.REJ_GROUP:
		return "incompatible group"
	case srt.REJX_BAD_REQUEST:
		return "general syntax error in the SocketID specification"
	case srt.REJX_UNAUTHORIZED:
		return "authentication failed"
	case srt.REJX_OVERLOAD:
		return "server too heavily loaded or exceeded credits"
	case srt.REJX_FORBIDDEN:
		return "access denied to the resource"
	case srt.REJX_NOTFOUND:
		return "resource not found at this time"
	case srt.REJX_BAD_MODE:
		return "specified mode not supported for this request"
	case srt.REJX_UNACCEPTABLE:
		return "requested parameters cannot be satisfied"
	case srt.REJX_CONFLICT:
		return "resource is already locked for modification"
	case srt.REJX_NOTSUP_MEDIA:
		return "media type not supported by the application"
	case srt.REJX_LOCKED:
		return "resource is locked for any access"
	case srt.REJX_FAILED_DEPEND:
		return "specified dependent session ID has been disconnected"
	case srt.REJX_ISE:
		return "unexpected internal server error"
	case srt.REJX_UNIMPLEMENTED:
		return "request recognized but not supported in current version"
	case srt.REJX_GW:
		return "gateway target endpoint rejected the connection"
	case srt.REJX_DOWN:
		return "service temporarily unavailable"
	case srt.REJX_VERSION:
		return "SRT version not supported"
	case srt.REJX_NOROOM:
		return "insufficient storage space for data stream"
	default:
		return fmt.Sprintf("unknown rejection code: %d", reason)
	}
}

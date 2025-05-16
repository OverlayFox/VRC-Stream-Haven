package srt

import (
	"errors"

	"github.com/OverlayFox/VRC-Stream-Haven/src/mediaServers/types"
	globalTypes "github.com/OverlayFox/VRC-Stream-Haven/src/types"
	vJson "github.com/OverlayFox/VRC-Stream-Haven/src/types/json"
	srt "github.com/datarhei/gosrt"
	"github.com/rs/zerolog"
)

type SRTServer struct {
	listener srt.Listener

	governor globalTypes.Governor

	logger zerolog.Logger

	quit chan struct{}
}

var _ globalTypes.ProtocolServer = (*SRTServer)(nil)

func New(listener srt.Listener, governor globalTypes.Governor, logger zerolog.Logger) (globalTypes.ProtocolServer, error) {
	return &SRTServer{
		listener: listener,

		governor: governor,

		logger: logger,
		quit:   make(chan struct{}),
	}, nil
}

// Start start the SRT server and blocks until it is closed or an error occurs.
func (s *SRTServer) Start() error {
	acceptCh := make(chan srt.ConnRequest)
	errCh := make(chan error)

	go func() {
		for {
			req, err := s.listener.Accept2()
			if err != nil {
				if errors.Is(err, srt.ErrListenerClosed) {
					s.logger.Debug().Msg("SRT server listener has been closed. Cannot accept more requests.")
					return
				}
				errCh <- err
				return
			}
			acceptCh <- req
		}
	}()
	s.logger.Info().Msgf("Started SRT listener on UDP address '%s'", s.listener.Addr().String())

	go func() {
		for {
			select {
			case <-s.quit:
				return

			case req := <-acceptCh:
				s.logger.Debug().Msgf("Received SRT connection request from '%s'", req.RemoteAddr().String())

				srtConnType, reason, streamRequest, haven := s.handleConnectionRequest(req)
				if srtConnType == srt.REJECT {
					s.logger.Info().
						Str("reject_reason", rejectionReasonToString(reason)).
						Msgf("Denied SRT connection request for '%s'", req.RemoteAddr().String())
					req.Reject(reason)
					continue
				}

				conn, err := req.Accept()
				if err != nil {
					s.logger.Error().
						Err(err).
						Msgf("Failed to accept connection request for '%s'", req.RemoteAddr().String())
					req.Reject(srt.REJ_UNKNOWN)
					continue
				}

				go s.handleConnection(conn, haven, streamRequest)
				s.logger.Debug().Msgf("Accepted SRT connection request for '%s'", conn.RemoteAddr().String())

			case err := <-errCh:
				s.logger.Error().Err(err).Msg("Failed to accept connection request")
			}
		}
	}()

	return nil
}

// handleConnectionRequest processes an incoming SRT connection request and determines
// whether to accept or reject the connection based on various criteria such as SRT version,
// encryption, stream ID, and connection type. It returns the connection type and a rejection
// reason if applicable.
func (s *SRTServer) handleConnectionRequest(req srt.ConnRequest) (srt.ConnType, srt.RejectionReason, types.StreamRequest, globalTypes.Haven) {
	streamIdRequest := req.StreamId()

	if req.Version() != 5 {
		s.logger.Debug().
			Str("client_ip", req.RemoteAddr().String()).
			Uint32("client_version", req.Version()).
			Msg("Clients SRT Version is not supported. Needs to be version 5")
		return srt.REJECT, srt.REJ_VERSION, types.StreamRequest{}, nil
	}

	streamRequest, err := types.NewStreamRequestFromStreamId(streamIdRequest)
	if err != nil {
		s.logger.Debug().
			Str("client_ip", req.RemoteAddr().String()).
			Err(err).
			Msg("Failed to parse clients stream-id")
		return srt.REJECT, srt.REJX_BAD_MODE, types.StreamRequest{}, nil
	}

	if !req.IsEncrypted() {
		s.logger.Debug().
			Str("client_ip", req.RemoteAddr().String()).
			Msg("Clients provided stream is not encrypted")
		return srt.REJECT, srt.REJ_UNSECURE, types.StreamRequest{}, nil
	}

	haven, err := s.governor.GetHaven(streamRequest.StreamId)
	if err != nil {
		s.logger.Debug().
			Str("client_ip", req.RemoteAddr().String()).
			Err(err).
			Msg("Clients requested haven does not exist")
		return srt.REJECT, srt.REJ_UNKNOWN, types.StreamRequest{}, nil
	}

	err = req.SetPassphrase(haven.GetPassphrase())
	if err != nil {
		s.logger.Debug().
			Str("client_ip", req.RemoteAddr().String()).
			Err(err).
			Msg("Clients provided stream key is incorrect for haven")
		return srt.REJECT, srt.REJ_BADSECRET, types.StreamRequest{}, nil
	}

	flagship := haven.GetFlagship()
	if streamRequest.ConnectionType == types.ConnectionTypeFlagship {
		if flagship != nil {
			s.logger.Debug().
				Str("client_ip", req.RemoteAddr().String()).
				Msg("Another client is already publishing to the requested haven")
			return srt.REJECT, srt.REJX_CONFLICT, types.StreamRequest{}, nil
		}
		return srt.PUBLISH, srt.REJ_UNKNOWN, streamRequest, haven

	} else if streamRequest.ConnectionType == types.ConnectionTypeEscort {
		if flagship == nil {
			s.logger.Debug().
				Str("client_ip", req.RemoteAddr().String()).
				Str("stream_id", req.StreamId()).
				Msg("Clients requested haven does not have a flagship")
			return srt.REJECT, srt.REJX_FAILED_DEPEND, types.StreamRequest{}, nil
		}
	} else {
		s.logger.Debug().
			Str("client_ip", req.RemoteAddr().String()).
			Msg("Client is using a connection type that is not yet supported")
		return srt.REJECT, srt.REJX_BAD_MODE, types.StreamRequest{}, nil
	}

	_, err = haven.GetPacketBuffer()
	if err != nil {
		if errors.Is(err, globalTypes.ErrBufferNotReady) {
			s.logger.Debug().
				Str("client_ip", req.RemoteAddr().String()).
				Str("stream_id", req.StreamId()).
				Err(err).
				Msg("Havens buffer is not ready")
			return srt.REJECT, srt.REJ_RESOURCE, types.StreamRequest{}, nil
		}

		s.logger.Error().
			Str("client_ip", req.RemoteAddr().String()).
			Str("stream_id", req.StreamId()).
			Err(err).
			Msg("Clients requested haven buffer is not available for an unknown reason")

		return srt.REJECT, srt.REJ_SYSTEM, types.StreamRequest{}, nil
	}
	return srt.SUBSCRIBE, srt.REJ_UNKNOWN, streamRequest, haven
}

// closeConnection closes the SRT connection and logs any errors that occur.
func (s *SRTServer) closeConnection(conn srt.Conn) {
	if err := conn.Close(); err != nil {
		s.logger.Error().
			Str("client_ip", conn.RemoteAddr().String()).
			Err(err).
			Msg("Failed to close the connection")
	}
}

// handleConnection handles an accepted SRT connection
func (s *SRTServer) handleConnection(conn srt.Conn, haven globalTypes.Haven, streamRequest types.StreamRequest) {
	if streamRequest.ConnectionType == types.ConnectionTypeFlagship {
		go s.handleFlagship(conn, haven)
	} else {
		go s.handleEscort(conn, announce, reqHaven)
	}
}

func (s *SRTServer) handleFlagship(conn srt.Conn, reqHaven globalTypes.Haven) {
	defer func() {
		s.closeConnection(conn)
	}()

	packageBuffer, err := reqHaven.GetPacketBuffer()
	if err != nil {
		s.logger.Error().
			Str("client_ip", conn.RemoteAddr().String()).
			Str("stream_id", conn.StreamId()).
			Err(err).
			Msg("No package buffer is available for the flagship to publish to")
	}

	sessionLogger := s.logger.With().
		Str("client_ip", conn.RemoteAddr().String()).
		Str("stream_id", conn.StreamId()).
		Bool("is_flagship", true).Logger()
	session := NewMediaSession(conn, types.ConnectionTypeFlagship, packageBuffer, sessionLogger)
	err = reqHaven.AddFlagship(session)
	if err != nil {
		s.logger.Error().
			Str("client_ip", conn.RemoteAddr().String()).
			Str("stream_id", conn.StreamId()).
			Err(err).
			Msg("Failed to add client as publisher to stream")
		return
	}

	session.ReadFromSession()
}

func (s *SRTServer) handleEscort(conn srt.Conn, announce vJson.Announce, reqHaven globalTypes.Haven) {
	defer func() {
		s.closeConnection(conn)
	}()

	packageBuffer, err := reqHaven.GetPacketBuffer()
	if err != nil {
		s.logger.Error().
			Str("client_ip", conn.RemoteAddr().String()).
			Str("stream_id", conn.StreamId()).
			Err(err).
			Msg("No flagship is available for the haven or the buffer is not ready")
		return
	}

	sessionLogger := s.logger.With().
		Str("client_ip", conn.RemoteAddr().String()).
		Str("stream_id", conn.StreamId()).
		Bool("is_publisher", false).Logger()

	session := NewMediaSession(conn, types.ConnectionTypeEscort, packageBuffer, sessionLogger)
	reqHaven.AddEscort(session)

	session.WriteToSession()
}

func (s *SRTServer) Close() {
	s.logger.Info().Msg("Closing SRT server")
	s.listener.Close()
	close(s.quit)

	s.listener = nil
	s.governor = nil
}

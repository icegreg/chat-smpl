// Package fsdirectory provides FreeSWITCH directory integration via mod_xml_curl
package fsdirectory

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// UserInfo represents user data for FreeSWITCH authentication
type UserInfo struct {
	ID          string
	Username    string
	Extension   string
	SIPPassword string // Dedicated SIP/Verto password
	DisplayName string
}

// Handler handles FreeSWITCH mod_xml_curl requests
type Handler struct {
	pool   *pgxpool.Pool
	domain string
	logger *zap.Logger
}

// NewHandler creates a new FreeSWITCH directory handler
func NewHandler(pool *pgxpool.Pool, domain string, logger *zap.Logger) *Handler {
	return &Handler{
		pool:   pool,
		domain: domain,
		logger: logger,
	}
}

// ServeHTTP handles POST requests from FreeSWITCH mod_xml_curl
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeNotFound(w)
		return
	}

	// Parse form data from FreeSWITCH
	if err := r.ParseForm(); err != nil {
		h.logger.Error("failed to parse form", zap.Error(err))
		h.writeNotFound(w)
		return
	}

	section := r.FormValue("section")
	h.logger.Debug("FreeSWITCH directory request",
		zap.String("section", section),
		zap.Any("form", r.Form))

	switch section {
	case "directory":
		h.handleDirectory(w, r)
	default:
		h.writeNotFound(w)
	}
}

// handleDirectory handles directory lookups for user authentication
func (h *Handler) handleDirectory(w http.ResponseWriter, r *http.Request) {
	purpose := r.FormValue("purpose")
	user := r.FormValue("user")
	domain := r.FormValue("domain")

	h.logger.Debug("directory lookup",
		zap.String("purpose", purpose),
		zap.String("user", user),
		zap.String("domain", domain))

	// Look up user by extension (the "user" in Verto is the extension)
	userInfo, err := h.getUserByExtension(r.Context(), user)
	if err != nil {
		h.logger.Warn("user not found", zap.String("extension", user), zap.Error(err))
		h.writeNotFound(w)
		return
	}

	h.logger.Info("user found for FreeSWITCH auth",
		zap.String("extension", userInfo.Extension),
		zap.String("username", userInfo.Username))

	h.writeUserXML(w, userInfo)
}

// getUserByExtension looks up user by their extension number
func (h *Handler) getUserByExtension(ctx context.Context, extension string) (*UserInfo, error) {
	query := `
		SELECT id::text, username, extension, COALESCE(sip_password, ''), COALESCE(display_name, username)
		FROM con_test.users
		WHERE extension = $1
	`

	var info UserInfo
	err := h.pool.QueryRow(ctx, query, extension).Scan(
		&info.ID,
		&info.Username,
		&info.Extension,
		&info.SIPPassword,
		&info.DisplayName,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user with extension %s not found", extension)
		}
		return nil, err
	}

	return &info, nil
}

// getUserByUsername looks up user by username (for login with username instead of extension)
func (h *Handler) getUserByUsername(ctx context.Context, username string) (*UserInfo, error) {
	query := `
		SELECT id::text, username, extension, COALESCE(sip_password, ''), COALESCE(display_name, username)
		FROM con_test.users
		WHERE username = $1 AND extension IS NOT NULL
	`

	var info UserInfo
	err := h.pool.QueryRow(ctx, query, username).Scan(
		&info.ID,
		&info.Username,
		&info.Extension,
		&info.SIPPassword,
		&info.DisplayName,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user %s not found", username)
		}
		return nil, err
	}

	return &info, nil
}

// writeUserXML writes the FreeSWITCH directory XML response for a user
func (h *Handler) writeUserXML(w http.ResponseWriter, user *UserInfo) {
	// FreeSWITCH directory XML format
	xmlResp := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<document type="freeswitch/xml">
  <section name="directory">
    <domain name="%s">
      <params>
        <param name="dial-string" value="{presence_id=${dialed_user}@${dialed_domain}}${sofia_contact(${dialed_user}@${dialed_domain})}"/>
      </params>
      <groups>
        <group name="default">
          <users>
            <user id="%s">
              <params>
                <param name="password" value="%s"/>
                <param name="vm-password" value="%s"/>
                <param name="jsonrpc-allowed-methods" value="verto"/>
                <param name="jsonrpc-allowed-event-channels" value="demo,conference,presence"/>
              </params>
              <variables>
                <variable name="user_id" value="%s"/>
                <variable name="user_context" value="default"/>
                <variable name="effective_caller_id_name" value="%s"/>
                <variable name="effective_caller_id_number" value="%s"/>
              </variables>
            </user>
          </users>
        </group>
      </groups>
    </domain>
  </section>
</document>`,
		h.domain,
		user.Extension,   // User ID is the extension
		user.SIPPassword, // Dedicated SIP password for Verto auth
		user.Extension,
		user.ID,
		xmlEscape(user.DisplayName),
		user.Extension,
	)

	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(xmlResp))
}

// writeNotFound writes a "not found" XML response
func (h *Handler) writeNotFound(w http.ResponseWriter) {
	xmlResp := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<document type="freeswitch/xml">
  <section name="result">
    <result status="not found"/>
  </section>
</document>`

	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(xmlResp))
}

// xmlEscape escapes special XML characters
func xmlEscape(s string) string {
	var buf strings.Builder
	_ = xml.EscapeText(&buf, []byte(s))
	return buf.String()
}

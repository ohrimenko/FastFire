package riststore

import (
	"bytes"
	"encoding/base32"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"

	"github.com/dgraph-io/ristretto"
)

// Amount of time for cookies/ristretto keys to expire.
var sessionExpire = 86400 * 30

// SessionSerializer provides an interface hook for alternative serializers
type SessionSerializer interface {
	Deserialize(d []byte, ss *sessions.Session) error
	Serialize(ss *sessions.Session) ([]byte, error)
}

// JSONSerializer encode the session map to JSON.
type JSONSerializer struct{}

// Serialize to JSON. Will err if there are unmarshalable key values
func (s JSONSerializer) Serialize(ss *sessions.Session) ([]byte, error) {
	m := make(map[string]interface{}, len(ss.Values))
	for k, v := range ss.Values {
		ks, ok := k.(string)
		if !ok {
			err := fmt.Errorf("Non-string key value, cannot serialize session to JSON: %v", k)
			fmt.Printf("riststore.JSONSerializer.serialize() Error: %v", err)
			return nil, err
		}
		m[ks] = v
	}
	return json.Marshal(m)
}

// Deserialize back to map[string]interface{}
func (s JSONSerializer) Deserialize(d []byte, ss *sessions.Session) error {
	m := make(map[string]interface{})
	err := json.Unmarshal(d, &m)
	if err != nil {
		fmt.Printf("riststore.JSONSerializer.deserialize() Error: %v", err)
		return err
	}
	for k, v := range m {
		ss.Values[k] = v
	}
	return nil
}

// GobSerializer uses gob package to encode the session map
type GobSerializer struct{}

// Serialize using gob
func (s GobSerializer) Serialize(ss *sessions.Session) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(ss.Values)
	if err == nil {
		return buf.Bytes(), nil
	}
	return nil, err
}

// Deserialize back to map[interface{}]interface{}
func (s GobSerializer) Deserialize(d []byte, ss *sessions.Session) error {
	dec := gob.NewDecoder(bytes.NewBuffer(d))
	return dec.Decode(&ss.Values)
}

// RistStore stores sessions in a ristretto backend.
type RistStore struct {
	Cache         *ristretto.Cache
	Codecs        []securecookie.Codec
	Options       *sessions.Options // default configuration
	DefaultMaxAge int               // default Redis TTL for a MaxAge == 0 session
	maxLength     int
	keyPrefix     string
	serializer    SessionSerializer
}

// SetMaxLength sets RistStore.maxLength if the `l` argument is greater or equal 0
// maxLength restricts the maximum length of new sessions to l.
// If l is 0 there is no limit to the size of a session, use with caution.
// The default for a new RistStore is 4096. Redis allows for max.
// Default: 4096,
func (s *RistStore) SetMaxLength(l int) {
	if l >= 0 {
		s.maxLength = l
	}
}

// SetKeyPrefix set the prefix
func (s *RistStore) SetKeyPrefix(p string) {
	s.keyPrefix = p
}

// SetSerializer sets the serializer
func (s *RistStore) SetSerializer(ss SessionSerializer) {
	s.serializer = ss
}

// SetMaxAge restricts the maximum age, in seconds, of the session record
// both in database and a browser. This is to change session storage configuration.
// If you want just to remove session use your session `s` object and change it's
// `Options.MaxAge` to -1, as specified in
//    http://godoc.org/github.com/gorilla/sessions#Options
//
// Default is the one provided by this package value - `sessionExpire`.
// Set it to 0 for no restriction.
// Because we use `MaxAge` also in SecureCookie crypting algorithm you should
// use this function to change `MaxAge` value.
func (s *RistStore) SetMaxAge(v int) {
	var c *securecookie.SecureCookie
	var ok bool
	s.Options.MaxAge = v
	for i := range s.Codecs {
		if c, ok = s.Codecs[i].(*securecookie.SecureCookie); ok {
			c.MaxAge(v)
		} else {
			fmt.Printf("Can't change MaxAge on codec %v\n", s.Codecs[i])
		}
	}
}

// NewRistStore returns a new RistStore.
func NewRistStore(c *ristretto.Cache, keyPairs ...[]byte) (*RistStore, error) {
	rs := &RistStore{
		Cache:  c,
		Codecs: securecookie.CodecsFromPairs(keyPairs...),
		Options: &sessions.Options{
			Path:   "/",
			MaxAge: sessionExpire,
		},
		DefaultMaxAge: 60 * 60, // 20 minutes seems like a reasonable default
		maxLength:     10485760,
		keyPrefix:     "session_",
		serializer:    GobSerializer{},
	}
	_, err := rs.ping()
	return rs, err
}

// Close closes the underlying *ristretto.Cache
func (s *RistStore) Close() error {
	s.Cache.Close()
	return nil
}

// Get returns a session for the given name after adding it to the registry.
//
// See gorilla/sessions FilesystemStore.Get().
func (s *RistStore) Get(r *http.Request, name string) (*sessions.Session, error) {
	return sessions.GetRegistry(r).Get(s, name)
}

// New returns a session for the given name without adding it to the registry.
//
// See gorilla/sessions FilesystemStore.New().
func (s *RistStore) New(r *http.Request, name string) (*sessions.Session, error) {
	var (
		err error
		ok  bool
	)
	session := sessions.NewSession(s, name)
	// make a copy
	options := *s.Options
	session.Options = &options
	session.IsNew = true
	if c, errCookie := r.Cookie(name); errCookie == nil {
		err = securecookie.DecodeMulti(name, c.Value, &session.ID, s.Codecs...)
		if err == nil {
			ok, err = s.load(session)
			session.IsNew = !(err == nil && ok) // not new if no error and data available
		}
	}
	return session, err
}

// Save adds a single session to the response.
func (s *RistStore) Save(r *http.Request, w http.ResponseWriter, session *sessions.Session) error {
	// Marked for deletion.
	if session.Options.MaxAge <= 0 {
		if err := s.delete(session); err != nil {
			return err
		}
		http.SetCookie(w, sessions.NewCookie(session.Name(), "", session.Options))
	} else {
		// Build an alphanumeric key for the ristretto store.
		if session.ID == "" {
			session.ID = strings.TrimRight(base32.StdEncoding.EncodeToString(securecookie.GenerateRandomKey(32)), "=")
		}
		if err := s.save(session); err != nil {
			return err
		}
		encoded, err := securecookie.EncodeMulti(session.Name(), session.ID, s.Codecs...)
		if err != nil {
			return err
		}
		http.SetCookie(w, sessions.NewCookie(session.Name(), encoded, session.Options))
	}
	return nil
}

// Delete removes the session from ristretto, and sets the cookie to expire.
//
// WARNING: This method should be considered deprecated since it is not exposed via the gorilla/sessions interface.
// Set session.Options.MaxAge = -1 and call Save instead. - July 18th, 2013
func (s *RistStore) Delete(r *http.Request, w http.ResponseWriter, session *sessions.Session) error {
	s.Cache.Del(s.keyPrefix + session.ID)
	// Set cookie to expire.
	options := *session.Options
	options.MaxAge = -1
	http.SetCookie(w, sessions.NewCookie(session.Name(), "", &options))
	// Clear session values.
	for k := range session.Values {
		delete(session.Values, k)
	}
	return nil
}

// ping does an internal ping against a server to check if it is alive.
func (s *RistStore) ping() (bool, error) {
	return true, nil
}

// save stores the session in ristretto.
func (s *RistStore) save(session *sessions.Session) error {
	b, err := s.serializer.Serialize(session)
	if err != nil {
		return err
	}
	if s.maxLength != 0 && len(b) > s.maxLength {
		return errors.New("SessionStore: the value to store is too big")
	}

	age := session.Options.MaxAge
	if age == 0 {
		age = s.DefaultMaxAge
	}
	if s.Cache.Set(s.keyPrefix+session.ID, b, int64(age)) {
		return nil
	} else {
		return errors.New("Fail save session in ristretto")
	}
}

// load ristretto the session from ristretto.
// returns true if there is a sessoin data in DB
func (s *RistStore) load(session *sessions.Session) (bool, error) {
	data, found := s.Cache.Get(s.keyPrefix + session.ID)
	if !found {
		return false, errors.New(s.keyPrefix + session.ID + " not in Session in ristretto")
	}
	if data == nil {
		return false, nil // no data was associated with this key
	}
	b, err := bytes_by_data(data)
	if err != nil {
		return false, err
	}
	return true, s.serializer.Deserialize(b, session)
}

// delete removes keys from ristretto if MaxAge<0
func (s *RistStore) delete(session *sessions.Session) error {
	s.Cache.Del(s.keyPrefix + session.ID)
	return nil
}

func bytes_by_data(reply interface{}) ([]byte, error) {
	switch reply := reply.(type) {
	case []byte:
		return reply, nil
	case string:
		return []byte(reply), nil
	case nil:
		return nil, errors.New("Data is nil")
	}
	return nil, fmt.Errorf("redigo: unexpected type for Bytes, got type %T", reply)
}

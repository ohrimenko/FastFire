package webrtc

import (
	"crypto/tls"
	"log"
	"net"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"syscall"

	"github.com/pion/turn/v2"
)

type TurnServer struct {
	IP       string
	PortUdp  int
	PortTcp  int
	PortTls  int
	Users    string
	Realm    string
	CertFile string
	KeyFile  string
	UsersMap map[string][]byte

	Turn *turn.Server
}

func NewTurnServer(ip string, portUdp int, portTcp int, portTls int, users string, realm string, certFile string, keyFile string) *TurnServer {
	return &TurnServer{
		IP:       ip,
		PortUdp:  portUdp,
		PortTcp:  portTcp,
		PortTls:  portTls,
		Users:    users,
		Realm:    realm,
		CertFile: certFile,
		KeyFile:  keyFile,
		UsersMap: map[string][]byte{},
	}
}

func (ts *TurnServer) Run() {
	if len(ts.IP) == 0 {
		log.Println("'public-ip' is required")
		return
	}

	if ts.Realm == "" {
		ts.Realm = "pion.ly"
	}

	turnServerConfig := turn.ServerConfig{}

	turnServerConfig.Realm = ts.Realm
	// PacketConnConfigs is a list of UDP Listeners and the configuration around them
	turnServerConfig.PacketConnConfigs = []turn.PacketConnConfig{}
	// ListenerConfig is a list of Listeners and the configuration around them
	turnServerConfig.ListenerConfigs = []turn.ListenerConfig{}

	if ts.PortUdp > 0 {
		// Create a UDP listener to pass into pion/turn
		// pion/turn itself doesn't allocate any UDP sockets, but lets the user pass them in
		// this allows us to add logging, storage or modify inbound/outbound traffic
		udpListener, err := net.ListenPacket("udp4", "0.0.0.0:"+strconv.Itoa(ts.PortUdp))
		if err != nil {
			log.Println("Failed to create TURN server listener: %s", err)
			return
		}

		turnServerConfig.PacketConnConfigs = append(turnServerConfig.PacketConnConfigs, turn.PacketConnConfig{
			PacketConn: udpListener,
			RelayAddressGenerator: &turn.RelayAddressGeneratorStatic{
				RelayAddress: net.ParseIP(ts.IP), // Claim that we are listening on IP passed by user (This should be your Public IP)
				Address:      "0.0.0.0",          // But actually be listening on every interface
			},
		})
	}

	if ts.PortTcp > 0 {
		// Create a TCP listener to pass into pion/turn
		// pion/turn itself doesn't allocate any TCP listeners, but lets the user pass them in
		// this allows us to add logging, storage or modify inbound/outbound traffic
		tcpListener, err := net.Listen("tcp4", "0.0.0.0:"+strconv.Itoa(ts.PortTcp))
		if err != nil {
			log.Println("Failed to create TURN server listener: %s", err)
			return
		}

		turnServerConfig.ListenerConfigs = append(turnServerConfig.ListenerConfigs, turn.ListenerConfig{
			Listener: tcpListener,
			RelayAddressGenerator: &turn.RelayAddressGeneratorStatic{
				RelayAddress: net.ParseIP(ts.IP),
				Address:      "0.0.0.0",
			},
		})
	}

	if ts.PortTls > 0 && len(ts.CertFile) > 0 && len(ts.KeyFile) > 0 {
		cer, err := tls.LoadX509KeyPair(ts.CertFile, ts.KeyFile)
		if err != nil {
			log.Println(err)
			return
		}

		// Create a TLS listener to pass into pion/turn
		// pion/turn itself doesn't allocate any TLS listeners, but lets the user pass them in
		// this allows us to add logging, storage or modify inbound/outbound traffic
		tlsListener, err := tls.Listen("tcp4", "0.0.0.0:"+strconv.Itoa(ts.PortTls), &tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{cer},
		})
		if err != nil {
			log.Println(err)
			return
		}

		turnServerConfig.ListenerConfigs = append(turnServerConfig.ListenerConfigs, turn.ListenerConfig{
			Listener: tlsListener,
			RelayAddressGenerator: &turn.RelayAddressGeneratorStatic{
				RelayAddress: net.ParseIP(ts.IP),
				Address:      "0.0.0.0",
			},
		})
	}

	if len(ts.Users) > 0 {
		// Cache -users flag for easy lookup later
		// If passwords are stored they should be saved to your DB hashed using turn.GenerateAuthKey
		for _, kv := range regexp.MustCompile(`(\w+)=(\w+)`).FindAllStringSubmatch(ts.Users, -1) {
			ts.UsersMap[kv[1]] = turn.GenerateAuthKey(kv[1], ts.Realm, kv[2])
		}

		// Set AuthHandler callback
		// This is called every time a user tries to authenticate with the TURN server
		// Return the key for that user, or false when no user is found
		turnServerConfig.AuthHandler = func(username string, realm string, srcAddr net.Addr) ([]byte, bool) {
			if key, ok := ts.UsersMap[username]; ok {
				return key, true
			}
			return nil, false
		}
	} else {
		//log.Println("'users' is required")
		//return
	}

	s, err := turn.NewServer(turnServerConfig)
	if err != nil {
		log.Println(err)
		return
	}

	ts.Turn = s

	// Block until user sends SIGINT or SIGTERM
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGQUIT)
	<-sigs

	if err = s.Close(); err != nil {
		log.Println(err)
		return
	}
}

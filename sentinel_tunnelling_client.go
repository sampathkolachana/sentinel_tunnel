package main

import (
	// "bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"sentinel_tunnel/st_logger"
	"sentinel_tunnel/st_sentinel_connection"
	"sync"
	"time"
)

type SentinelTunnellingDbConfig struct {
	Name       string
	Local_port string
}

type SentinelTunnellingConfiguration struct {
	Sentinels_addresses_list []string
	Databases                []SentinelTunnellingDbConfig
}

type SentinelTunnellingClient struct {
	configuration       SentinelTunnellingConfiguration
	sentinel_connection *st_sentinel_connection.Sentinel_connection
}

type get_db_address_by_name_function func(db_name string) (string, error)

func NewSentinelTunnellingClient(config_file_location string) *SentinelTunnellingClient {
	data, err := ioutil.ReadFile(config_file_location)
	if err != nil {
		st_logger.WriteLogMessage(st_logger.FATAL, "an error has occur during configuration read",
			err.Error())
	}

	Tunnelling_client := SentinelTunnellingClient{}
	err = json.Unmarshal(data, &(Tunnelling_client.configuration))
	if err != nil {
		st_logger.WriteLogMessage(st_logger.FATAL, "an error has occur during configuration read,",
			err.Error())
	}

	Tunnelling_client.sentinel_connection, err =
		st_sentinel_connection.NewSentinelConnection(Tunnelling_client.configuration.Sentinels_addresses_list)
	if err != nil {
		st_logger.WriteLogMessage(st_logger.FATAL, "an error has occur, ",
			err.Error())
	}

	st_logger.WriteLogMessage(st_logger.INFO, "done initializing Tunnelling")

	return &Tunnelling_client
}

func createTunnelling_old(conn1 net.Conn, conn2 net.Conn) {
	io.Copy(conn1, conn2)
	conn1.Close()
	conn2.Close()
}

func createTunnelling(conn1 net.Conn, conn2 net.Conn) {
	defer conn1.Close()
	defer conn2.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(conn1, conn2)
		// Signal peer that no more data is coming.
		conn1.Close()
	}()
	go func() {
		defer wg.Done()
		io.Copy(conn2, conn1)
		// Signal peer that no more data is coming.
		conn2.Close()
	}()

	wg.Wait()
}

func handleConnection(c net.Conn, db_name string,
	get_db_address_by_name get_db_address_by_name_function) {
	st_logger.WriteLogMessage(st_logger.INFO, "getting master address for db ", db_name, "...")
	db_address, err := get_db_address_by_name(db_name)
	if err != nil {
		st_logger.WriteLogMessage(st_logger.ERROR, "cannot get db address for ", db_name,
			",", err.Error())
		c.Close()
		return
	}
	st_logger.WriteLogMessage(st_logger.INFO, "got master address for db ", db_name, " : ", db_address)
	st_logger.WriteLogMessage(st_logger.INFO, "connecting to db_address ", db_address, "...")
	db_conn, err := net.Dial("tcp", db_address)
	if err != nil {
		st_logger.WriteLogMessage(st_logger.ERROR, "cannot connect to db ", db_name,
			",", err.Error())
		c.Close()
		return
	}
	st_logger.WriteLogMessage(st_logger.INFO, "tunnelling client to master db")
	go createTunnelling(c, db_conn)
	// go createTunnelling(db_conn, c)
}

func handleSigleDbConnections(listening_port string, db_name string,
	get_db_address_by_name get_db_address_by_name_function) {

	listener, err := net.Listen("tcp6", fmt.Sprintf(":%s", listening_port))
	if err != nil {
		st_logger.WriteLogMessage(st_logger.FATAL, "cannot listen to port ",
			listening_port, err.Error())
	}

	st_logger.WriteLogMessage(st_logger.INFO, "listening on port ", listening_port,
		" for connections to database: ", db_name)
	for {
		conn, err := listener.Accept()
		if err != nil {
			st_logger.WriteLogMessage(st_logger.FATAL, "cannot accept connections on port ",
				listening_port, err.Error())
		}
		st_logger.WriteLogMessage(st_logger.INFO, "accepted connection on port ", listening_port)
		go handleConnection(conn, db_name, get_db_address_by_name)
	}

}

func (st_client *SentinelTunnellingClient) Start() {
	for _, db_conf := range st_client.configuration.Databases {
		go handleSigleDbConnections(db_conf.Local_port, db_conf.Name,
			st_client.sentinel_connection.GetAddressByDbName)
	}
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage : sentinel_tunnel <config_file_path> <log_file_path>")
		return
	}
	st_logger.InitializeLogger(os.Args[2])
	st_client := NewSentinelTunnellingClient(os.Args[1])
	st_client.Start()
	for {
		time.Sleep(1000 * time.Millisecond)
	}
}

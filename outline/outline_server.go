package outline

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

var unsafeClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

// {
// "id":"3",
// "name":"",
// "password":"5PgTilMvdrhK",
// "port":61081,
// "method":"chacha20-ietf-poly1305",
// "accessUrl":"ss://Y2hhY2hhMjAtaWV0Zi1wb2x5MTMwNTo1UGdUaWxNdmRyaEs=@18.182.68.185:61081/?outline=1"
// }
type OutlineUser struct {
	ID               string      `json:"id"`
	JSID             template.JS `json:"_,omitempty"`
	Name             string      `json:"name"`
	Password         string      `json:"password"`
	Port             int         `json:"port"`
	Method           string      `json:"method"`
	AccessURL        string      `json:"accessUrl"`
	TransferredBytes ByteNum     `json:"byteNum,omitempty"`

	// provided by go manager
	IP       net.IP    `json:"_,omitempty"`
	Enabled  bool      `json:"_,omitempty"`
	EnColor  string    `json:"_,omitempty"`
	Online   bool      `json:"_,omitempty"`
	OnColor  string    `json:"_,omitempty"`
	DaysLeft int       `json:"_,omitempty"`
	Limit    int       `json:"_,omitempty"`
	Expire   string    `json:"_,omitempty"`
}

// Provide by Go Program
type GoUser struct {
	ID       string `json:"id"`
	IP       net.IP `json:"ip"`
	Enabled  bool   `json:"enabled"`
	Online   bool   `json:"online"`
	DaysLeft int    `json:"days_left"`
	Limit    int    `json:"limit"`
}

// Outline apiUrl
// https://127.0.0.1:56298/QQR9pcgCRP_g5OLX3n-w-g
type OutlineServer struct {
	ID    uint32  `json:"_,omitempty"`
	URL   string  `json:"_,omitempty"`
	GoURL string  `json:"_,omitempty"`
	Total ByteNum `json:"_,omitempty"`

	Name                 string `json:"name"`
	ServerID             string `json:"serverId"`
	MetricsEnabled       bool   `json:"metricsEnabled"`
	CreatedTimestampMs   uint64 `json:"createdTimestampMs"`
	PortForNewAccessKeys int    `json:"portForNewAccessKeys"`

	sync.Mutex `json:"_,omitempty"`
	logger     *zap.Logger             `json:"_,omitempty"`
	Users      map[string]*OutlineUser `json:"_,omitempty"`
}

func NewOutlineServer(id uint32, server string, l *zap.Logger) *OutlineServer {
	uri, _ := url.Parse(server)
	n, _ := strconv.Atoi(uri.Port())
	uri.Host = uri.Hostname()+":"+strconv.Itoa(n+1)
	uri.Scheme = "http"
	s := &OutlineServer{
		ID:     id,
		URL:    server,
		GoURL:  uri.String(),
		logger: l,
		Users:  make(map[string]*OutlineUser),
	}
	return s
}

func (s *OutlineServer) GetServerInfo() error {
	req, err := http.NewRequest(http.MethodGet, s.URL+"/server", nil)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	r, err := unsafeClient.Do(req)
	if err != nil {
		return err
	}
	if r.StatusCode != http.StatusOK {
		return errors.New("status code error, code: " + strconv.Itoa(r.StatusCode))
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	if err := json.Unmarshal(b, s); err != nil {
		return err
	}

	return nil
}

// AddUser: curl -X POST baseurl
func (s *OutlineServer) AddUser() (*OutlineUser, error) {
	s.logger.Info("add new user")
	req, err := http.NewRequest(http.MethodPost, s.URL+"/access-keys", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	r, err := unsafeClient.Do(req)
	if err != nil {
		return nil, err
	}
	if r.StatusCode != http.StatusCreated {
		return nil, errors.New("status code error, code: " + strconv.Itoa(r.StatusCode))
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	user := &OutlineUser{}
	if err := json.Unmarshal(b, user); err != nil {
		return nil, err
	}

	return user, nil
}

// DeleteUser: curl -X DELETE baseurl?id=1
func (s *OutlineServer) DeleteUser(id string) error {
	s.logger.Info(fmt.Sprintf("delete user %v", id))
	req, err := http.NewRequest(http.MethodDelete, s.URL+"/access-keys/"+id, nil)
	if err != nil {
		return err
	}

	r, err := unsafeClient.Do(req)
	if err != nil {
		return err
	}
	if r.StatusCode != http.StatusNoContent {
		return errors.New("status code error, code: " + strconv.Itoa(r.StatusCode))
	}

	return nil
}

// SetAllowance: curl -X PUT baseurl?id=1&allowance=50
func (s *OutlineServer) SetAllowance(id, num string) error {
	s.logger.Info(fmt.Sprintf("set user %v allowance to %v", id, num))
	n, err := strconv.Atoi(num)
	if err != nil {
		return err
	}
	type Limit struct {
		Bytes uint64 `json:"bytes"`
	}
	type Limitor struct {
		Limit `json:"limit"`
	}
	limit := Limitor{Limit: Limit{Bytes: uint64(n)<<30}}
	b, err := json.Marshal(limit)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPut, s.URL+"/access-keys/"+id+"/data-limit", bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	r, err := unsafeClient.Do(req)
	if err != nil {
		return err
	}
	if r.StatusCode != http.StatusNoContent {
		switch r.StatusCode {
		case http.StatusBadRequest:
			return errors.New("invalid data limit")
		case http.StatusNotFound:
			return errors.New("access key inexistent")
		}
		return errors.New("status code error, code: " + strconv.Itoa(r.StatusCode))
	}
	return nil
}

// RenameUser: curl -X PUT baseurl?id=1&name=test1
func (s *OutlineServer) RenameUser(id, n string) error {
	s.logger.Info(fmt.Sprintf("rename user %v name to %v", id, n))
	type Name struct {
		Name string `json:"name"`
	}

	name := Name{Name: n}
	b, err := json.Marshal(&name)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPut, s.URL+"/access-keys/"+id+"/name", bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	r, err := unsafeClient.Do(req)
	if err != nil {
		return err
	}
	if r.StatusCode != http.StatusNoContent {
		return errors.New("status code error, code: " + strconv.Itoa(r.StatusCode))
	}

	return nil
}

// api with customized outline vpn server
// change key status and set deadline of key
func (s *OutlineServer) ChangeGoUserStatus(id string) error {
	s.logger.Info(fmt.Sprintf("change go user %v status", id))

	req, err := http.NewRequest(http.MethodPatch, s.GoURL+"/go/manager?id="+id, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	r, err := unsafeClient.Do(req)
	if err != nil {
		return err
	}
	if r.StatusCode != http.StatusOK {
		return errors.New("status code error, code: " + strconv.Itoa(r.StatusCode))
	}

	return nil
}

func (s *OutlineServer) SetGoUserDeadline(id, days string) error {
	s.logger.Info(fmt.Sprintf("set go user %v deadline to %v days", id, days))

	req, err := http.NewRequest(http.MethodPut, s.GoURL+"/go/manager?id="+id+"&deadline="+days, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	r, err := unsafeClient.Do(req)
	if err != nil {
		return err
	}
	if r.StatusCode != http.StatusOK {
		return errors.New("status code error, code: " + strconv.Itoa(r.StatusCode))
	}

	return nil
}

func (s *OutlineServer) SetGoDataLimit(id, num string) error {
	s.logger.Info(fmt.Sprintf("set go user %v data limit to %v", id, num))

	req, err := http.NewRequest(http.MethodPost, s.GoURL+"/go/manager?id="+id+"&limit="+num, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	r, err := unsafeClient.Do(req)
	if err != nil {
		return err
	}
	if r.StatusCode != http.StatusOK {
		return errors.New("status code error, code: " + strconv.Itoa(r.StatusCode))
	}

	return nil
}

func (s *OutlineServer) GetGoUser() ([]*GoUser, error) {
	req, err := http.NewRequest(http.MethodGet, s.GoURL+"/go/manager", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	r, err := unsafeClient.Do(req)
	if err != nil {
		return nil, err
	}
	if r.StatusCode != http.StatusOK {
		return nil, errors.New("status code error, code: " + strconv.Itoa(r.StatusCode))
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	type Status struct {
		Status []*GoUser `json:"status"`
	}

	status := Status{}
	if err := json.Unmarshal(b, &status); err != nil {
		return nil, err
	}

	return status.Status, nil
}

func (s *OutlineServer) GetUsage() (map[string]uint64, error) {
	req, err := http.NewRequest(http.MethodGet, s.URL+"/metrics/transfer", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	r, err := unsafeClient.Do(req)
	if err != nil {
		return nil, err
	}
	if r.StatusCode != http.StatusOK {
		return nil, errors.New("status code error, code: " + strconv.Itoa(r.StatusCode))
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	type BytesTransferred struct {
		ByUserID map[string]uint64 `json:"bytesTransferredByUserId"`
	}

	used := BytesTransferred{ByUserID: make(map[string]uint64)}
	if err := json.Unmarshal(b, &used); err != nil {
		return nil, err
	}

	return used.ByUserID, nil
}

// GetAllUser: curl -X GET baseurl
func (s *OutlineServer) GetAllUser() error {
	s.logger.Info("get all users info")
	s.Lock()
	s.Users = make(map[string]*OutlineUser)
	s.Unlock()

	usage, err := s.GetUsage()
	if err != nil {
		return err
	}
	goUser, err := s.GetGoUser()
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodGet, s.URL+"/access-keys", nil)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	r, err := unsafeClient.Do(req)
	if err != nil {
		return err
	}
	if r.StatusCode != http.StatusOK {
		return errors.New("status code error, code: " + strconv.Itoa(r.StatusCode))
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	type Users struct {
		AccessKeys []*OutlineUser `json:"accessKeys"`
	}

	users := Users{}
	if err := json.Unmarshal(b, &users); err != nil {
		return err
	}
	s.Lock()
	for _, user := range users.AccessKeys {
		n := usage[user.ID]
		user.TransferredBytes = ByteNum(n)
		s.Users[user.ID] = user
	}
	for _, user := range goUser {
		if usr, ok := s.Users[user.ID]; ok {
			usr.IP = user.IP
			usr.Enabled = user.Enabled
			if user.Enabled {
				usr.EnColor = "green"
			} else {
				usr.EnColor = "red"
			}
			usr.Online = user.Online
			if user.Online {
				usr.OnColor = "green"
			} else {
				usr.OnColor = "red"
			}
			usr.DaysLeft = user.DaysLeft
			usr.Limit = user.Limit
		}
	}
	now := time.Now()
	for _, usr := range s.Users {
		if usr.EnColor == "" {
			usr.Enabled = true
		}
		usr.AccessURL = strings.TrimSuffix(usr.AccessURL, "/?outline=1") + "#YnamlyVPN"
		usr.Expire = now.Add(time.Hour * 24 * time.Duration(usr.DaysLeft)).Format("2006-01-02")
	}
	s.Unlock()

	return nil
}

func (s *OutlineServer) SetRouter(prefix string, r *http.ServeMux) {
	// baseurl GET
	// GetAllUsers
	r.HandleFunc(prefix, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.HandlerFunc(http.NotFound).ServeHTTP(w, r)
			return
		}

		if err := s.GetAllUser(); err != nil {
			s.logger.Error(fmt.Sprintf("get all user error: %v", err))
		}
		s.Lock()
		users := make([]*OutlineUser, 0, len(s.Users))
		for _, user := range s.Users {
			user.JSID = template.JS("\"" + user.ID + "\"")
			users = append(users, user)
		}
		s.Unlock()

		sort.Slice(users, func(i, j int) bool {
			if len(users[i].ID) < len(users[j].ID) {
				return true
			}
			if len(users[i].ID) > len(users[j].ID) {
				return false
			}
			return users[i].ID < users[j].ID
		})

		type Info struct {
			Server *OutlineServer
			Users  []*OutlineUser
		}
		info := Info{Server: s, Users: users}
		usage, err := s.GetUsage()
		if err != nil {
			s.logger.Error(fmt.Sprintf("get all user usage: %v", err))
			return
		}
		s.Lock()
		info.Server.Total = 0
		for _, v := range usage {
			info.Server.Total += ByteNum(v)
		}
		s.Unlock()
		if err := serverPanelTemplate.Execute(w, info); err != nil {
			s.logger.Error(fmt.Sprintf("template error: %v", err))
		}
	})

	// baseurl POST
	// AddUser
	r.HandleFunc(prefix+"/user", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.HandlerFunc(http.NotFound).ServeHTTP(w, r)
			return
		}

		user, err := s.AddUser()
		if err != nil {
			s.logger.Error(fmt.Sprintf("add new user error: %v", err))
			http.HandlerFunc(http.NotFound).ServeHTTP(w, r)
			return
		}
		if err := s.SetGoUserDeadline(user.ID, "30"); err != nil {
			s.logger.Error(fmt.Sprintf("set go user deadline error: %v", err))
			http.HandlerFunc(http.NotFound).ServeHTTP(w, r)
			return
		}
	})

	// baseurl?id={id} DELETE
	// delete user from server
	r.HandleFunc(prefix+"/id", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.HandlerFunc(http.NotFound).ServeHTTP(w, r)
			return
		}

		id := r.URL.Query().Get("id")
		if id == "" {
			http.HandlerFunc(http.NotFound).ServeHTTP(w, r)
			return
		}
		if err := s.DeleteUser(id); err != nil {
			s.logger.Error(fmt.Sprintf("delete user error: %v", err))
			http.HandlerFunc(http.NotFound).ServeHTTP(w, r)
			return
		}
		s.Lock()
		delete(s.Users, id)
		s.Unlock()
	})

	// baseurl?id={id}&name={name} PUT
	// rename a user
	r.HandleFunc(prefix+"/name", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.HandlerFunc(http.NotFound).ServeHTTP(w, r)
			return
		}

		id := r.URL.Query().Get("id")
		name := r.URL.Query().Get("name")
		if id == "" {
			http.HandlerFunc(http.NotFound).ServeHTTP(w, r)
			return
		}
		if err := s.RenameUser(id, name); err != nil {
			s.logger.Error(fmt.Sprintln("rename user error: %v", err))
			http.HandlerFunc(http.NotFound).ServeHTTP(w, r)
			return
		}
	})

	// baseurl?id={id}?allowance={usage} PUT
	// update key data allowance
	r.HandleFunc(prefix+"/data", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.HandlerFunc(http.NotFound).ServeHTTP(w, r)
			return
		}

		id := r.URL.Query().Get("id")
		allowance := r.URL.Query().Get("allowance")
		if id == "" || allowance == "" {
			http.HandlerFunc(http.NotFound).ServeHTTP(w, r)
			return
		}
		if err := s.SetGoDataLimit(id, allowance); err != nil {
			s.logger.Error(fmt.Sprintln("set go user allowance error: %v", err))
			http.HandlerFunc(http.NotFound).ServeHTTP(w, r)
			return
		}
		if err := s.SetAllowance(id, allowance); err != nil {
			s.logger.Error(fmt.Sprintln("set user allowance error: %v", err))
			http.HandlerFunc(http.NotFound).ServeHTTP(w, r)
			return
		}
	})

	// baseurl?id={id}
	// change user from enabled to disbled or vice verse
	r.HandleFunc(prefix+"/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			http.HandlerFunc(http.NotFound).ServeHTTP(w, r)
			return
		}

		id := r.URL.Query().Get("id")
		if id == "" {
			http.HandlerFunc(http.NotFound).ServeHTTP(w, r)
			return
		}
		if err := s.ChangeGoUserStatus(id); err != nil {
			s.logger.Error(fmt.Sprintln("change go user status error: %v", err))
			http.HandlerFunc(http.NotFound).ServeHTTP(w, r)
			return
		}
	})

	// baseurl?id={id}&time={days}
	// set up the last day of this account
	r.HandleFunc(prefix+"/deadline", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.HandlerFunc(http.NotFound).ServeHTTP(w, r)
			return
		}

		id := r.URL.Query().Get("id")
		days := r.URL.Query().Get("days")
		if id == "" || days == "" {
			http.HandlerFunc(http.NotFound).ServeHTTP(w, r)
			return
		}
		if err := s.SetGoUserDeadline(id, days); err != nil {
			s.logger.Error(fmt.Sprintln("set go user left days error: %v", err))
			http.HandlerFunc(http.NotFound).ServeHTTP(w, r)
			return
		}
	})
}

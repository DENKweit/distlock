package main

import (
	"encoding/json"
	"flag"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi"
	"github.com/lucsky/cuid"

	"github.com/DENKweit/distlock/types"
)

type session struct {
	ID    string `json:"id"`
	Key   string `json:"-"`
	Timer *time.Timer
}

type lockableValue struct {
	Value     string
	IsLocked  bool
	SessionID *string
}

func startTimer(duration time.Duration, s *session, lock *sync.RWMutex, kvs map[string]*lockableValue, sessions map[string]*session) {
	if s.Timer != nil {
		s.Timer.Stop()
	}
	s.Timer = time.AfterFunc(duration, func() {
		lock.Lock()
		defer lock.Unlock()

		delete(kvs, s.Key)
		delete(sessions, s.ID)
	})
}

func main() {

	var port int
	flag.IntVar(&port, "port", 9876, "set port")
	flag.Parse()

	router := chi.NewRouter()

	sessions := map[string]*session{}
	kvLock := sync.RWMutex{}
	kvs := map[string]*lockableValue{}

	locksLock := sync.Mutex{}
	locks := map[string]chan struct{}{}

	router.Get("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(types.StatusReturn{Running: true})
	})

	router.Post("/session/renew/{sessionId}/{duration}", func(w http.ResponseWriter, r *http.Request) {
		kvLock.Lock()
		defer kvLock.Unlock()

		sessionId := chi.URLParam(r, "sessionId")
		duration := chi.URLParam(r, "duration")

		interval, err := strconv.ParseInt(duration, 10, 64)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if session, ok := sessions[sessionId]; ok {
			startTimer(time.Duration(interval), session, &kvLock, kvs, sessions)
		}
	})

	router.Post("/session/destroy/{sessionId}", func(w http.ResponseWriter, r *http.Request) {
		kvLock.Lock()
		defer kvLock.Unlock()

		sessionId := chi.URLParam(r, "sessionId")
		if session, ok := sessions[sessionId]; ok {
			session.Timer.Stop()
			delete(kvs, session.Key)
			delete(sessions, sessionId)
		}
	})

	router.Get("/kv/keys", func(w http.ResponseWriter, r *http.Request) {
		prefix := r.URL.Query().Get("prefix")
		kvLock.RLock()

		ret := []string{}
		for key := range kvs {
			if prefix != "" && key[:len(prefix)] == prefix {
				ret = append(ret, key)
			} else if prefix == "" {
				ret = append(ret, key)
			}
		}

		kvLock.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ret)
	})

	router.Post("/kv/acquire/{key}/{duration}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		kvLock.Lock()

		key := chi.URLParam(r, "key")
		duration := chi.URLParam(r, "duration")

		interval, err := strconv.ParseInt(duration, 10, 64)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		value := r.URL.Query().Get("value")

		ret := types.AcquireReturn{
			SessionID: cuid.New(),
			Success:   false,
		}

		if _, ok := kvs[key]; !ok {
			kvs[key] = &lockableValue{
				Value:     value,
				IsLocked:  false,
				SessionID: &ret.SessionID,
			}
		}

		if !kvs[key].IsLocked {

			kvs[key].IsLocked = true

			sessions[ret.SessionID] = &session{
				ID:  ret.SessionID,
				Key: key,
			}

			if kvs[key].SessionID == nil {
				kvs[key].SessionID = &ret.SessionID
			}

			startTimer(time.Duration(interval), sessions[ret.SessionID], &kvLock, kvs, sessions)

			ret.Success = true
		}

		kvLock.Unlock()
		json.NewEncoder(w).Encode(ret)
	})

	router.Post("/kv/release/{key}/{sessionId}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		key := chi.URLParam(r, "key")
		sessionID := chi.URLParam(r, "sessionId")

		kvLock.Lock()

		ret := types.ReleaseReturn{
			Success: false,
		}

		if v, ok := kvs[key]; ok {
			if session, sessionOk := sessions[sessionID]; sessionOk && session.Key == key {
				v.IsLocked = false
				ret.Success = true
			}
		}

		kvLock.Unlock()
		json.NewEncoder(w).Encode(ret)
		return

	})

	router.Post("/mutex/lock/{key}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		key := chi.URLParam(r, "key")
		timeoutStr := r.URL.Query().Get("timeout")
		var timeout *time.Duration

		if timeoutStr != "" {
			interval, err := strconv.ParseInt(timeoutStr, 10, 64)

			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			if interval <= 0 {
				http.Error(w, "timeout must be >= 0", http.StatusBadRequest)
				return
			}

			t := time.Duration(interval)

			timeout = &t
		}

		var currentMutex chan struct{}

		locksLock.Lock()

		if m, ok := locks[key]; ok {
			currentMutex = m
		} else {
			locks[key] = make(chan struct{}, 1)
			currentMutex = locks[key]
		}

		locksLock.Unlock()

		if timeout == nil {
			select {
			case currentMutex <- struct{}{}:
			}
		} else {
			select {
			case currentMutex <- struct{}{}:
			case <-time.After(*timeout):
			}
		}

		ret := types.MutexReturn{
			Success: true,
		}

		json.NewEncoder(w).Encode(ret)
	})

	router.Post("/mutex/unlock/{key}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		key := chi.URLParam(r, "key")

		locksLock.Lock()

		if m, ok := locks[key]; ok {
			if len(m) == 0 {
				locksLock.Unlock()
				http.Error(w, "can't unlock unlocked mutex", http.StatusBadRequest)
				return
			} else {
				<-m
				locksLock.Unlock()
				ret := types.MutexReturn{
					Success: true,
				}
				json.NewEncoder(w).Encode(ret)
				return
			}
		}

		locksLock.Unlock()
		http.Error(w, "mutex does not exist", http.StatusBadRequest)
		return
	})

	router.Post("/int/{key}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		key := chi.URLParam(r, "key")
		sessionId := r.URL.Query().Get("sessionId")
		op := r.URL.Query().Get("op")
		value := r.URL.Query().Get("value")

		ret := types.IntReturn{
			Success: false,
			Op:      op,
		}

		kvLock.Lock()

		if op != string(types.IntOpTypeGet) {
			if v, ok := kvs[key]; ok {
				if v.SessionID != nil && *v.SessionID != sessionId {
					kvLock.Unlock()
					json.NewEncoder(w).Encode(ret)
					return
				}
			}
		}

		if _, ok := kvs[key]; !ok {
			kvs[key] = &lockableValue{
				Value:    strconv.FormatInt(0, 10),
				IsLocked: false,
			}
		}

		switch op {
		case string(types.IntOpTypeInc):
			if kvs[key].Value == "" {
				kvs[key].Value = "0"
			}
			currentValue, err := strconv.ParseInt(kvs[key].Value, 10, 64)
			if err != nil {
				kvLock.Unlock()
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			currentValue++
			ret.Value = currentValue
			ret.Success = true
			kvs[key].Value = strconv.FormatInt(currentValue, 10)
		case string(types.IntOpTypeDec):
			if kvs[key].Value == "" {
				kvs[key].Value = "0"
			}
			currentValue, err := strconv.ParseInt(kvs[key].Value, 10, 64)
			if err != nil {
				kvLock.Unlock()
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			currentValue--
			ret.Success = true
			ret.Value = currentValue
			kvs[key].Value = strconv.FormatInt(currentValue, 10)
		case string(types.IntOpTypeGet):
			if kvs[key].Value == "" {
				kvs[key].Value = "0"
			}
			currentValue, err := strconv.ParseInt(kvs[key].Value, 10, 64)
			if err != nil {
				kvLock.Unlock()
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			ret.Success = true
			ret.Value = currentValue
		case string(types.IntOpTypeSet):
			currentValue, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				kvLock.Unlock()
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			ret.Value = currentValue
			ret.Success = true
			kvs[key].Value = strconv.FormatInt(currentValue, 10)
		}

		kvLock.Unlock()
		json.NewEncoder(w).Encode(ret)
	})

	router.Post("/kv/set/{key}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		key := chi.URLParam(r, "key")
		sessionId := r.URL.Query().Get("sessionId")
		value := r.URL.Query().Get("value")

		kvLock.Lock()

		ret := types.SetReturn{
			Success: false,
		}

		if sessionId != "" {
			if session, sessionOk := sessions[sessionId]; sessionOk && session.Key == key {
				kvs[key].Value = value
				ret.Success = true
			}
		} else {
			if _, ok := kvs[key]; !ok {
				kvs[key] = &lockableValue{
					Value:    value,
					IsLocked: false,
				}
				ret.Success = true
			}
		}

		kvLock.Unlock()
		json.NewEncoder(w).Encode(ret)
		return
	})

	router.Get("/kv/get/{key}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		key := chi.URLParam(r, "key")

		kvLock.RLock()

		ret := types.GetReturn{
			Success: false,
		}

		if v, ok := kvs[key]; ok {
			ret.Success = true
			ret.Key = key
			ret.Value = v.Value
		}

		kvLock.RUnlock()
		json.NewEncoder(w).Encode(ret)
		return
	})

	router.Get("/kv/getm", func(w http.ResponseWriter, r *http.Request) {

		req := &types.GetMRequest{}
		err := json.NewDecoder(r.Body).Decode(&req)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		ret := types.GetMReturn{
			Entries: make([]types.KeyValueSuccess, len(req.Keys)),
		}

		kvLock.RLock()

		for idx, key := range req.Keys {
			v, ok := kvs[key]
			if ok {
				ret.Entries[idx].Success = true
				ret.Entries[idx].Value = v.Value
			}
			ret.Entries[idx].Key = key
		}

		kvLock.RUnlock()

		json.NewEncoder(w).Encode(ret)
		return
	})

	router.Post("/kv/setm", func(w http.ResponseWriter, r *http.Request) {
		req := &types.SetMRequest{}
		err := json.NewDecoder(r.Body).Decode(&req)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		sessionID := r.URL.Query().Get("sessionId")

		ret := types.SetMReturn{
			Success: false,
		}

		kvLock.Lock()

		for _, entry := range req.Entries {
			if v, ok := kvs[entry.Key]; ok {
				if v.IsLocked {
					if sessionID == "" || (v.SessionID != nil && *v.SessionID != sessionID) {
						kvLock.Unlock()
						json.NewEncoder(w).Encode(ret)
						return
					}
				}
			}
		}

		for _, entry := range req.Entries {
			kvs[entry.Key] = &lockableValue{Value: entry.Value, IsLocked: false}
		}

		kvLock.Unlock()

		ret.Success = true

		json.NewEncoder(w).Encode(ret)
		return
	})

	err := http.ListenAndServe(":9876", router)
	if err != nil {
		panic(err)
	}
}

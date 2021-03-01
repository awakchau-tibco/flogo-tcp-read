package tcpread

import (
	"github.com/project-flogo/core/data/coerce"
)

// Settings ...
type Settings struct {
	Network         string `md:"network"`       // The network type
	Host            string `md:"host"`          // The host name or IP for TCP server.
	Port            string `md:"port,required"` // The port to listen on
	TimeoutMs       int64  `md:"timeoutMs"`     // Timeout for tcp read operation in milliseconds
	Delimiter       string `md:"delimiter"`     // Data delimiter for read and write
	CustomDelimiter string `md:"customDelimiter"`
}

// ToMap ...
func (s *Settings) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"network":         s.Network,
		"host":            s.Host,
		"port":            s.Port,
		"timeoutMs":       s.TimeoutMs,
		"delimiter":       s.Delimiter,
		"customDelimiter": s.CustomDelimiter,
	}
}

// FromMap ...
func (s *Settings) FromMap(values map[string]interface{}) error {
	var err error
	s.Network, err = coerce.ToString(values["network"])
	if err != nil {
		return err
	}
	s.Host, err = coerce.ToString(values["host"])
	if err != nil {
		return err
	}
	s.Port, err = coerce.ToString(values["port"])
	if err != nil {
		return err
	}
	s.TimeoutMs, err = coerce.ToInt64(values["timeoutMs"])
	if err != nil {
		return err
	}
	s.Delimiter, err = coerce.ToString(values["delimiter"])
	if err != nil {
		return err
	}
	s.CustomDelimiter, err = coerce.ToString(values["customDelimiter"])
	if err != nil {
		return err
	}
	return nil
}

// HandlerSettings ...
type HandlerSettings struct {
}

// Output ...
type Output struct {
	Data string `md:"data"` // The data received from the connection
}

// Reply ///
type Reply struct {
	Reply string `md:"reply"` // The reply to be sent back
}

// ToMap ...
func (o *Output) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"data": o.Data,
	}
}

// FromMap ...
func (o *Output) FromMap(values map[string]interface{}) error {

	var err error
	o.Data, err = coerce.ToString(values["data"])
	if err != nil {
		return err
	}

	return nil
}

// ToMap ...
func (r *Reply) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"reply": r.Reply,
	}
}

// FromMap ...
func (r *Reply) FromMap(values map[string]interface{}) error {

	var err error
	r.Reply, err = coerce.ToString(values["reply"])
	if err != nil {
		return err
	}

	return nil
}

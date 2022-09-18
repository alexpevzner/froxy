// Froxy - HTTP over SSH proxy
//
// Copyright (C) 2019 and up by Alexander Pevzner (pzz@apevzner.com)
// See LICENSE for license terms and conditions
//
// IDN support

package main

import (
	"bytes"
	"encoding/json"
	"strings"

	"golang.org/x/net/idna"
)

//
// IDN version of ServerParams
//
type IDNServerParams ServerParams

var _ = json.Marshaler(&IDNServerParams{})
var _ = json.Unmarshaler(&IDNServerParams{})

//
// Marshal IDNServerParams to JSON. Recodes host name to IDN on a fly
//
func (p *IDNServerParams) MarshalJSON() ([]byte, error) {
	out := ServerParams(*p)
	out.Addr = IDNDecode(out.Addr)
	return json.Marshal(out)
}

//
// Unarshal IDNServerParams from JSON. Recodes host name to IDN on a fly
//
func (p *IDNServerParams) UnmarshalJSON(data []byte) error {
	in := ServerParams{}
	err := json.Unmarshal(data, &in)
	if err != nil {
		return err
	}

	in.Addr = IDNEncode(in.Addr)
	*p = IDNServerParams(in)

	return nil
}

//
// IDN version of SiteParams
//
type IDNSiteParams SiteParams

var _ = json.Marshaler(&IDNSiteParams{})
var _ = json.Unmarshaler(&IDNSiteParams{})

//
// Marshal IDNSiteParams to JSON. Recodes host name to IDN on a fly
//
func (p *IDNSiteParams) MarshalJSON() ([]byte, error) {
	out := SiteParams(*p)
	out.Host = IDNDecode(out.Host)
	return json.Marshal(out)
}

//
// Unarshal IDNSiteParams from JSON. Recodes host name to IDN on a fly
//
func (p *IDNSiteParams) UnmarshalJSON(data []byte) error {
	in := SiteParams{}
	err := json.Unmarshal(data, &in)
	if err != nil {
		return err
	}

	in.Host = IDNEncode(in.Host)

	*p = IDNSiteParams(in)
	return nil
}

//
// IDN version of []SiteParams
//
type IDNSiteParamsList []SiteParams

var _ = json.Marshaler(IDNSiteParamsList(nil))

//
// Marshal IDNSiteParams to JSON. Recodes host name to IDN on a fly
//
func (list IDNSiteParamsList) MarshalJSON() ([]byte, error) {
	buf := bytes.Buffer{}

	buf.WriteByte('[')
	for i := range list {
		data, err := (*IDNSiteParams)(&list[i]).MarshalJSON()
		if err != nil {
			return nil, err
		}

		if i != 0 {
			buf.WriteByte(',')
		}
		buf.Write(data)
	}
	buf.WriteByte(']')

	return buf.Bytes(), nil
}

//
// Decode string from IDN to UNICODE
//
func IDNDecode(in string) string {
	out, err := idna.ToUnicode(in)
	if err == nil {
		return out
	} else {
		return in
	}
}

//
// Encode string from UNICODE to IDN
//
func IDNEncode(in string) string {
	out, err := idna.ToASCII(strings.ToLower(in))
	if err == nil {
		return out
	} else {
		return in
	}
}

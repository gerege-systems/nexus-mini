package oidc

import "encoding/base64"

func b64urlDecode(s string) ([]byte, error) { return base64.RawURLEncoding.DecodeString(s) }

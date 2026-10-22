package discovery

import (
	"crypto/sha256"
	"encoding/hex"
)

// TXT builds the TXT record of the _woodhouse._tcp advertisement; the keys are
// documented under "Discovery" in the API README.
//
//   - fp is always present: the lowercase hex SHA-256 of the API's TLS leaf
//     certificate (DER), so an app can recognise a server it has already pinned
//     whatever its address.
//   - setup=1 is present while an appliance (woodhouse-os) has no admin yet,
//     i.e. it is waiting to be adopted. Outside appliance mode the first admin
//     is created by the first login, so there is no setup state to announce.
func TXT(apiCertDER []byte, appliance, hasAdmin bool) []string {
	sum := sha256.Sum256(apiCertDER)
	txt := []string{"fp=" + hex.EncodeToString(sum[:])}
	if appliance && !hasAdmin {
		txt = append(txt, "setup=1")
	}
	return txt
}

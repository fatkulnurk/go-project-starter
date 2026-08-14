// Package id menyediakan generator identifier bersama. Ini satu-satunya
// tempat yang mengimpor library UUID; module dan platform memanggil fungsi di
// sini, bukan library UUID secara langsung, sehingga pemilihan implementasi
// tetap terpusat.
package id

import "github.com/google/uuid"

// New returns a version-7 UUID string. UUID v7 is time-ordered so rows sort
// naturally by insertion time, which keeps indexes friendly.
func New() string {
	v, err := uuid.NewV7()
	if err != nil {
		panic("id: uuid v7 unavailable: " + err.Error())
	}
	return v.String()
}

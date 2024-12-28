package cache

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type User struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// GetID gets the ID of the User
//
// implements core.Identifiable
func (user User) GetID() uuid.UUID {
	return user.ID
}

func TestCanEncryptData(t *testing.T) {
	encryptionKey := []byte("@v3ry#S3cr3tK3y!")
	cache := New[User]("test")
	cache.encryptionKey = encryptionKey

	encrypted, err := cache.encrypt([]byte("Hello, World!"))
	require.NoError(t, err, "Failed to encrypt the data")
	require.Greater(t, len(encrypted), 0, "The encrypted data is empty")
	require.NotEqual(t, "Hello, World!", string(encrypted), "The data is not encrypted")
}

func TestCanDecryptData(t *testing.T) {
	encryptionKey := []byte("@v3ry#S3cr3tK3y!")
	cache := New[User]("test")
	cache.encryptionKey = encryptionKey

	encrypted, err := cache.encrypt([]byte("Hello, World!"))
	require.NoError(t, err, "Failed to encrypt the data")
	require.Greater(t, len(encrypted), 0, "The encrypted data is empty")

	decrypted, err := cache.decrypt(encrypted)
	require.NoError(t, err, "Failed to decrypt the data")
	require.Equal(t, "Hello, World!", string(decrypted), "The data is not decrypted")
}

func TestShouldFailWithWrongEncryptedData(t *testing.T) {
	encryptionKey := []byte("@v3ry#S3cr3tK3y!")
	cache := New[User]("test")
	cache.encryptionKey = encryptionKey

	encrypted, err := cache.encrypt([]byte("Hello, World!"))
	require.NoError(t, err, "Failed to encrypt the data")
	require.Greater(t, len(encrypted), 0, "The encrypted data is empty")

	// Shorten the data (corrupt it)
	encrypted = encrypted[:10]
	_, err = cache.decrypt(encrypted)
	require.Error(t, err, "Decryption should have failed")
}

package testdump

import (
	"bytes"
	"os"

	"github.com/alextanhongpin/core/internal"
)

type fileReaderWriter struct {
	fileName string
}

func newFileReaderWriter(fileName string) *fileReaderWriter {
	return &fileReaderWriter{
		fileName: fileName,
	}
}

func (rw *fileReaderWriter) Read() ([]byte, error) {
	return os.ReadFile(rw.fileName)
}

func (rw *fileReaderWriter) Write(b []byte) error {
	return internal.WriteIfNotExists(rw.fileName, b)
}

type Hook[T any] func(S[T]) S[T]

func Snapshot[T any](rw readerWriter, t T, s S[T], hooks ...Hook[T]) error {
	// Run middleware in reverse order, so that the first
	// will execute first.
	for i := 0; i < len(hooks); i++ {
		mw := hooks[len(hooks)-i-1]
		s = mw(s)
	}

	b, err := s.Marshal(t)
	if err != nil {
		return err
	}

	if err := rw.Write(b); err != nil {
		return err
	}

	receivedBytes := bytes.Clone(b)
	// NOTE: We unmarshal back the bytes, since there might
	// be additional information not present during the
	// marshalling process.
	received, err := s.Unmarshal(b)
	if err != nil {
		return err
	}

	b, err = rw.Read()
	if err != nil {
		return err
	}

	snapshotBytes := bytes.Clone(b)
	snapshot, err := s.Unmarshal(b)
	if err != nil {
		return err
	}

	// This is required when comparing JSON/YAML type, because
	// unmarshalling the type to map[any]interface{} will cause
	// information to be lost (e.g. additional fields).
	sb, err := s.UnmarshalAny(snapshotBytes)
	if err != nil {
		return err
	}

	rb, err := s.UnmarshalAny(receivedBytes)
	if err != nil {
		return err
	}

	if err := s.CompareAny(sb, rb); err != nil {
		return err
	}

	return s.Compare(snapshot, received)
}

func MarshalHook[T any](hook func(T) (T, error)) Hook[T] {
	return func(s S[T]) S[T] {
		return &marshalHook[T]{
			S:    s,
			hook: hook,
		}
	}
}

func CompareHook[T any](hook func(snapshot T, received T) error) Hook[T] {
	return func(s S[T]) S[T] {
		return &compareHook[T]{
			S:    s,
			hook: hook,
		}
	}
}

type marshalHook[T any] struct {
	S[T]
	hook func(t T) (T, error)
}

func (m *marshalHook[T]) Marshal(t T) ([]byte, error) {
	if m.hook != nil {
		var err error
		t, err = m.hook(t)
		if err != nil {
			return nil, err
		}
	}
	return m.S.Marshal(t)
}

type compareHook[T any] struct {
	S[T]
	hook func(snapshot, received T) error
}

func (m *compareHook[T]) Compare(snapshot, received T) error {
	if m.hook != nil {
		if err := m.hook(snapshot, received); err != nil {
			return err
		}
	}

	return m.S.Compare(snapshot, received)
}

func Copier[T any]() Hook[T] {
	return func(s S[T]) S[T] {
		return &marshalHook[T]{
			S: s,
			hook: func(t T) (T, error) {
				return internal.Copy(t)
			},
		}
	}
}

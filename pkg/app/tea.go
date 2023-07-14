package app

import (
	"io"

	tea "github.com/charmbracelet/bubbletea"
)

type Tea struct {
	reads   *io.PipeReader
	writes  *io.PipeWriter
	program *tea.Program
}

var _ App = (*Tea)(nil)

// Return the handle that allows you to write to this stream.
func (s *Tea) Writer() io.Writer {
	return s.writes
}

// Resizing does nothing to a stream.
func (s *Tea) Resize(size Size) error {
	s.program.Send(tea.WindowSizeMsg{
		Width:  size.Columns,
		Height: size.Rows,
	})
	return nil
}

func (s *Tea) Write(data []byte) (n int, err error) {
	return s.writes.Write(data)
}

func (s *Tea) Read(p []byte) (n int, err error) {
	return s.reads.Read(p)
}

func NewTea(model tea.Model, size Size) *Tea {
	reads, out := io.Pipe()
	in, writes := io.Pipe()

	program := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithOutput(out),
		tea.WithInput(in),
	)

	go program.Run()

	tea := &Tea{
		reads:   reads,
		writes:  writes,
		program: program,
	}

	tea.Resize(size)

	return tea
}

package gosh

import (
	"testing"
)

func BenchmarkNewBuilderPattern(b *testing.B) {
	ConfigureGlobals()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := New().Arg("echo").Arg("benchmark test").Exec()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkArgsMethod(b *testing.B) {
	ConfigureGlobals()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := New().Args("echo", "benchmark", "args", "test").Exec()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCommandMethod(b *testing.B) {
	ConfigureGlobals()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := New().Command("echo").Arg("benchmark").Arg("command").Exec()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkComplexChaining(b *testing.B) {
	ConfigureGlobals()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := New().
			Command("echo").
			Arg("complex").
			Args("benchmark", "test").
			Env("BENCH_VAR", "test").
			Dir("/tmp").
			Exec()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLegacyConstructor(b *testing.B) {
	ConfigureGlobals()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := NewLegacy("echo", "legacy", "benchmark").Exec()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark builder creation without execution
func BenchmarkBuilderCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = New().
			Command("echo").
			Arg("test").
			Args("multiple", "args").
			Env("TEST", "value").
			Dir("/tmp")
	}
}

// Benchmark HTTP writer creation (without actual HTTP calls)
func BenchmarkHTTPWriterCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		writer := NewHTTPStreamWriter("http://localhost:8080/logs")
		_ = writer
	}
}

// Benchmark building shell without execution
func BenchmarkShellBuilding(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		shell := New()
		shell = shell.Command("ls")
		shell = shell.Arg("-la")
		shell = shell.Dir("/tmp")
		shell = shell.Env("TEST", "benchmark")
		_ = shell
	}
}

// Memory allocation benchmark
func BenchmarkMemoryAllocation(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		shell := New().
			Command("echo").
			Args("memory", "allocation", "test").
			Env("MEM_TEST", "value").
			Dir("/tmp")
		
		// Simulate some work without actual execution
		_ = shell.command
		_ = shell.args
		_ = shell.env
		_ = shell.dir
	}
}
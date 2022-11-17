// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build darwin || linux || windows
// +build darwin linux windows

// An app that draws a green triangle on a red background.
//
// In order to build this program as an Android APK, using the gomobile tool.
//
// See http://godoc.org/golang.org/x/mobile/cmd/gomobile to install gomobile.
//
// Get the basic example and use gomobile to build or install it on your device.
//
//	$ go get -d golang.org/x/mobile/example/basic
//	$ gomobile build golang.org/x/mobile/example/basic # will build an APK
//
//	# plug your Android device to your computer or start an Android emulator.
//	# if you have adb installed on your machine, use gomobile install to
//	# build and deploy the APK to an Android target.
//	$ gomobile install golang.org/x/mobile/example/basic
//
// Switch to your device or emulator to start the Basic application from
// the launcher.
// You can also run the application on your desktop by running the command
// below. (Note: It currently doesn't work on Windows.)
//
//	$ go install golang.org/x/mobile/example/basic && basic
package main

import (
	"encoding/binary"
	"log"

	"golang.org/x/mobile/app"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
	"golang.org/x/mobile/event/touch"
	"golang.org/x/mobile/exp/app/debug"
	"golang.org/x/mobile/exp/f32"
	"golang.org/x/mobile/exp/gl/glutil"
	"golang.org/x/mobile/gl"
)

var (
	images   *glutil.Images
	fps      *debug.FPS
	program  gl.Program
	position gl.Attrib
	offset   gl.Uniform
	scale    gl.Uniform
	aspect   gl.Uniform
	color    gl.Attrib
	buf      gl.Buffer

	locX     float32
	locY     float32
	difX     float32
	difY     float32
	ratio    float32
	koef     float32
	isMoving bool
)

func main() {
	app.Main(func(a app.App) {
		var glctx gl.Context
		var sz size.Event
		for e := range a.Events() {
			switch e := a.Filter(e).(type) {
			case lifecycle.Event:
				switch e.Crosses(lifecycle.StageVisible) {
				case lifecycle.CrossOn:
					glctx, _ = e.DrawContext.(gl.Context)
					onStart(glctx)
					a.Send(paint.Event{})
				case lifecycle.CrossOff:
					onStop(glctx)
					glctx = nil
				}
			case size.Event:
				sz = e
				locX = float32(sz.WidthPx / 2)
				locY = float32(sz.HeightPx / 2)
				ratio = float32(sz.HeightPx) / float32(sz.WidthPx)
			case paint.Event:
				if glctx == nil || e.External {
					// As we are actively painting as fast as
					// we can (usually 60 FPS), skip any paint
					// events sent by the system.
					continue
				}

				onPaint(glctx, sz)
				a.Publish()
				// Drive the animation by preparing to paint the next frame
				// after this one is shown.
				a.Send(paint.Event{})
			case touch.Event:
				if e.Type == touch.TypeBegin {
					difX = locX - e.X
					difY = locY - e.Y
					isMoving = true
				} else if e.Type == touch.TypeMove {
					if isMoving {
						locX = e.X + difX
						locY = e.Y + difY
					}
				} else if e.Type == touch.TypeEnd {
					difX = 0
					difY = 0
					isMoving = false
				}

			case key.Event:
				if e.Direction == key.DirNone || e.Direction == key.DirPress {
					switch e.Rune {
					case '+', '=':
						koef *= 1.1
					case '-':
						koef /= 1.1
					}
				}
			}
		}
	})
}

func onStart(glctx gl.Context) {
	koef = 1

	var err error
	program, err = glutil.CreateProgram(glctx, vertexShader, fragmentShader)
	if err != nil {
		log.Printf("error creating GL program: %v", err)
		return
	}

	buf = glctx.CreateBuffer()
	glctx.BindBuffer(gl.ARRAY_BUFFER, buf)
	glctx.BufferData(gl.ARRAY_BUFFER, triangleData, gl.STATIC_DRAW)

	position = glctx.GetAttribLocation(program, "position")
	color = glctx.GetAttribLocation(program, "color")
	offset = glctx.GetUniformLocation(program, "offset")
	aspect = glctx.GetUniformLocation(program, "aspect")
	scale = glctx.GetUniformLocation(program, "scale")

	images = glutil.NewImages(glctx)
	fps = debug.NewFPS(images)
}

func onStop(glctx gl.Context) {
	glctx.DeleteProgram(program)
	glctx.DeleteBuffer(buf)
	fps.Release()
	images.Release()
}

func onPaint(glctx gl.Context, sz size.Event) {
	glctx.ClearColor(0.2, 0.2, 0.2, 1)
	glctx.Clear(gl.COLOR_BUFFER_BIT)

	glctx.UseProgram(program)

	glctx.Uniform2f(offset, locX/float32(sz.WidthPx), locY/float32(sz.HeightPx))

	glctx.Uniform1f(aspect, ratio)
	glctx.Uniform1f(scale, koef)

	glctx.BindBuffer(gl.ARRAY_BUFFER, buf)
	glctx.EnableVertexAttribArray(position)
	glctx.EnableVertexAttribArray(color)
	vertexBytesSize := 4 * (coordsPerVertex + colorValuesPerVertex)
	colorOffset := 4 * coordsPerVertex
	glctx.VertexAttribPointer(position, coordsPerVertex, gl.FLOAT, false, vertexBytesSize, 0)
	glctx.VertexAttribPointer(color, colorValuesPerVertex, gl.FLOAT, false, vertexBytesSize, colorOffset)
	glctx.DrawArrays(gl.TRIANGLES, 0, vertexCount)
	glctx.DisableVertexAttribArray(position)
	glctx.DisableVertexAttribArray(color)

	fps.Draw(sz)
}

var triangleData = f32.Bytes(binary.LittleEndian,
	/*coords*/ 0.0, 0.3, 0.0 /*color*/, 1.0, 0.0, 0.0, 1.0, // top left
	/*coords*/ -0.26, -0.15, 0.0 /*color*/, 0.0, 1.0, 0.0, 1.0, // bottom left
	/*coords*/ 0.26, -0.15, 0.0 /*color*/, 0.0, 0.0, 1.0, 1.0, // bottom right
)

const (
	coordsPerVertex      = 3
	colorValuesPerVertex = 4
	vertexCount          = 3
)

const vertexShader = `#version 100
uniform vec2 offset;
uniform float aspect;
uniform float scale;
varying vec4 difColor;

attribute vec4 position;
attribute vec4 color;

void main() {
	// offset comes in with x/y values between 0 and 1.
	// position bounds are -1 to 1.
	vec4 offset4 = vec4(2.0*offset.x-1.0, 1.0-2.0*offset.y, 0, 0);
	vec4 scaledPos = vec4(position.x * aspect * scale, position.y * scale, position.z, position.w);
	gl_Position = scaledPos + offset4;
	difColor = color;
}
`

const fragmentShader = `#version 100
precision mediump float;
varying vec4 difColor;
void main() {
	gl_FragColor = difColor;
}`

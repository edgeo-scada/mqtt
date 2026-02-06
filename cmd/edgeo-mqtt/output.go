// Copyright 2025 Edgeo SCADA
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/edgeo-scada/mqtt/mqtt"
)

// OutputFormat represents the output format type.
type OutputFormat string

const (
	FormatTable OutputFormat = "table"
	FormatJSON  OutputFormat = "json"
	FormatCSV   OutputFormat = "csv"
	FormatRaw   OutputFormat = "raw"
)

// MessageOutput represents a message for output.
type MessageOutput struct {
	Timestamp  time.Time              `json:"timestamp,omitempty"`
	Topic      string                 `json:"topic"`
	Payload    string                 `json:"payload"`
	QoS        int                    `json:"qos"`
	Retain     bool                   `json:"retain"`
	PacketID   uint16                 `json:"packet_id,omitempty"`
	Properties *MessagePropsOutput    `json:"properties,omitempty"`
	Duplicate  bool                   `json:"duplicate,omitempty"`
}

// MessagePropsOutput represents MQTT 5.0 properties for output.
type MessagePropsOutput struct {
	ContentType     string            `json:"content_type,omitempty"`
	ResponseTopic   string            `json:"response_topic,omitempty"`
	CorrelationData string            `json:"correlation_data,omitempty"`
	UserProperties  map[string]string `json:"user_properties,omitempty"`
	MessageExpiry   *uint32           `json:"message_expiry,omitempty"`
}

// Formatter handles output formatting.
type Formatter struct {
	format     OutputFormat
	timestamps bool
	showProps  bool
	writer     io.Writer
	csvWriter  *csv.Writer
	first      bool
}

// NewFormatter creates a new formatter.
func NewFormatter(format string, timestamps, showProps bool) *Formatter {
	f := &Formatter{
		format:     OutputFormat(format),
		timestamps: timestamps,
		showProps:  showProps,
		writer:     os.Stdout,
		first:      true,
	}
	if f.format == FormatCSV {
		f.csvWriter = csv.NewWriter(os.Stdout)
	}
	return f
}

// FormatMessage formats and prints a message.
func (f *Formatter) FormatMessage(msg *mqtt.Message) {
	switch f.format {
	case FormatJSON:
		f.formatJSON(msg)
	case FormatCSV:
		f.formatCSV(msg)
	case FormatRaw:
		f.formatRaw(msg)
	default:
		f.formatTable(msg)
	}
}

func (f *Formatter) formatTable(msg *mqtt.Message) {
	var sb strings.Builder

	// Timestamp
	if f.timestamps {
		sb.WriteString(colorize(time.Now().Format("15:04:05.000"), ColorGray))
		sb.WriteString(" ")
	}

	// Topic
	sb.WriteString(colorize(msg.Topic, ColorCyan))
	sb.WriteString(" ")

	// QoS and Retain
	qosStr := fmt.Sprintf("QoS%d", msg.QoS)
	sb.WriteString(colorize("[", ColorGray))
	sb.WriteString(colorize(qosStr, ColorYellow))
	if msg.Retain {
		sb.WriteString(colorize(",R", ColorMagenta))
	}
	sb.WriteString(colorize("]", ColorGray))
	sb.WriteString(" ")

	// Payload
	payload := string(msg.Payload)
	if len(payload) > 200 && !verbose {
		payload = payload[:200] + "..."
	}
	sb.WriteString(payload)

	fmt.Fprintln(f.writer, sb.String())

	// Properties
	if f.showProps && msg.Properties != nil {
		f.printProperties(msg.Properties)
	}
}

func (f *Formatter) formatJSON(msg *mqtt.Message) {
	output := MessageOutput{
		Topic:   msg.Topic,
		Payload: string(msg.Payload),
		QoS:     int(msg.QoS),
		Retain:  msg.Retain,
	}

	if f.timestamps {
		output.Timestamp = time.Now()
	}

	if msg.PacketID > 0 {
		output.PacketID = msg.PacketID
	}

	if msg.Duplicate {
		output.Duplicate = true
	}

	if f.showProps && msg.Properties != nil {
		output.Properties = convertProperties(msg.Properties)
	}

	data, _ := json.Marshal(output)
	fmt.Fprintln(f.writer, string(data))
}

func (f *Formatter) formatCSV(msg *mqtt.Message) {
	if f.first {
		headers := []string{"topic", "payload", "qos", "retain"}
		if f.timestamps {
			headers = append([]string{"timestamp"}, headers...)
		}
		f.csvWriter.Write(headers)
		f.first = false
	}

	record := []string{
		msg.Topic,
		string(msg.Payload),
		fmt.Sprintf("%d", msg.QoS),
		fmt.Sprintf("%t", msg.Retain),
	}

	if f.timestamps {
		record = append([]string{time.Now().Format(time.RFC3339Nano)}, record...)
	}

	f.csvWriter.Write(record)
	f.csvWriter.Flush()
}

func (f *Formatter) formatRaw(msg *mqtt.Message) {
	fmt.Fprintln(f.writer, string(msg.Payload))
}

func (f *Formatter) printProperties(props *mqtt.Properties) {
	indent := "    "

	if props.ContentType != "" {
		fmt.Fprintf(f.writer, "%sContent-Type: %s\n", indent, props.ContentType)
	}
	if props.ResponseTopic != "" {
		fmt.Fprintf(f.writer, "%sResponse-Topic: %s\n", indent, props.ResponseTopic)
	}
	if len(props.CorrelationData) > 0 {
		fmt.Fprintf(f.writer, "%sCorrelation-Data: %s\n", indent, string(props.CorrelationData))
	}
	if props.MessageExpiry != nil {
		fmt.Fprintf(f.writer, "%sMessage-Expiry: %ds\n", indent, *props.MessageExpiry)
	}
	if props.PayloadFormat != nil {
		format := "bytes"
		if *props.PayloadFormat == 1 {
			format = "UTF-8"
		}
		fmt.Fprintf(f.writer, "%sPayload-Format: %s\n", indent, format)
	}
	for _, up := range props.UserProperties {
		fmt.Fprintf(f.writer, "%s%s: %s\n", indent, up.Key, up.Value)
	}
}

func convertProperties(props *mqtt.Properties) *MessagePropsOutput {
	if props == nil {
		return nil
	}

	out := &MessagePropsOutput{
		ContentType:    props.ContentType,
		ResponseTopic:  props.ResponseTopic,
		MessageExpiry:  props.MessageExpiry,
	}

	if len(props.CorrelationData) > 0 {
		out.CorrelationData = string(props.CorrelationData)
	}

	if len(props.UserProperties) > 0 {
		out.UserProperties = make(map[string]string)
		for _, up := range props.UserProperties {
			out.UserProperties[up.Key] = up.Value
		}
	}

	return out
}

// Color codes for terminal output.
const (
	ColorReset   = "\033[0m"
	ColorRed     = "\033[31m"
	ColorGreen   = "\033[32m"
	ColorYellow  = "\033[33m"
	ColorBlue    = "\033[34m"
	ColorMagenta = "\033[35m"
	ColorCyan    = "\033[36m"
	ColorGray    = "\033[90m"
	ColorBold    = "\033[1m"
)

// colorize wraps text with color codes.
func colorize(text, color string) string {
	if noColor {
		return text
	}
	return color + text + ColorReset
}

// TableWriter writes formatted tables.
type TableWriter struct {
	headers []string
	rows    [][]string
	widths  []int
}

// NewTableWriter creates a new table writer.
func NewTableWriter(headers ...string) *TableWriter {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	return &TableWriter{
		headers: headers,
		widths:  widths,
	}
}

// AddRow adds a row to the table.
func (t *TableWriter) AddRow(values ...string) {
	for i, v := range values {
		if i < len(t.widths) && len(v) > t.widths[i] {
			t.widths[i] = len(v)
		}
	}
	t.rows = append(t.rows, values)
}

// Render renders the table to stdout.
func (t *TableWriter) Render() {
	// Print header
	for i, h := range t.headers {
		fmt.Printf("%-*s  ", t.widths[i], colorize(h, ColorBold))
	}
	fmt.Println()

	// Print separator
	for i := range t.headers {
		fmt.Print(strings.Repeat("-", t.widths[i]) + "  ")
	}
	fmt.Println()

	// Print rows
	for _, row := range t.rows {
		for i, v := range row {
			if i < len(t.widths) {
				fmt.Printf("%-*s  ", t.widths[i], v)
			}
		}
		fmt.Println()
	}
}

// PrintKeyValue prints a key-value pair formatted nicely.
func PrintKeyValue(key, value string) {
	fmt.Printf("  %-20s %s\n", colorize(key+":", ColorCyan), value)
}

// PrintSection prints a section header.
func PrintSection(title string) {
	fmt.Printf("\n%s\n", colorize(title, ColorBold))
	fmt.Println(strings.Repeat("-", len(title)))
}

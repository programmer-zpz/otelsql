// Copyright Sam Xie
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

package otelsql

import (
	"context"
	"database/sql/driver"

	"go.opentelemetry.io/otel/trace"
)

var _ driver.Connector = (*otConnector)(nil)

type otConnector struct {
	driver.Connector
	otDriver *otDriver
}

func newConnector(connector driver.Connector, otDriver *otDriver) *otConnector {
	return &otConnector{
		Connector: connector,
		otDriver:  otDriver,
	}
}

func (c *otConnector) Connect(ctx context.Context) (connection driver.Conn, err error) {
	method := MethodConnectorConnect
	onDefer := recordMetric(ctx, c.otDriver.cfg.Instruments, c.otDriver.cfg.Attributes, method)
	defer func() {
		onDefer(err)
	}()

	var span trace.Span
	if c.otDriver.cfg.SpanOptions.AllowRoot || trace.SpanContextFromContext(ctx).IsValid() {
		ctx, span = c.otDriver.cfg.Tracer.Start(ctx, c.otDriver.cfg.SpanNameFormatter.Format(ctx, method, ""),
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(c.otDriver.cfg.Attributes...),
		)
		defer span.End()
	}

	connection, err = c.Connector.Connect(ctx)
	if err != nil {
		recordSpanError(span, c.otDriver.cfg.SpanOptions, err)
		return nil, err
	}
	return newConn(connection, c.otDriver.cfg), nil
}

func (c *otConnector) Driver() driver.Driver {
	return c.otDriver
}

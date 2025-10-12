package apideprecation

import (
	"google.golang.org/protobuf/reflect/protoreflect"
)

type fieldReporter struct {
	builder *planBuilder
}

func newFieldReporter(seedDesc []protoreflect.MessageDescriptor) *fieldReporter {
	return &fieldReporter{builder: newPlanBuilder(seedDesc)}
}

func (r *fieldReporter) Report(
	msg protoreflect.Message,
	meta callMeta,
	onDeprecatedField onDeprecatedFieldFunc,
	onDeprecatedEnum onDeprecatedEnumFunc,
) {
	plan := r.builder.LoadOrBuild(msg.Descriptor())
	plan.EvalMessage(msg, meta, onDeprecatedField, onDeprecatedEnum)
}

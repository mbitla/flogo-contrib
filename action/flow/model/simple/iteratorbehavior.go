package simple

import (
	"fmt"

	"github.com/TIBCOSoftware/flogo-contrib/action/flow/model"
	"github.com/TIBCOSoftware/flogo-lib/core/data"
)

// SimpleIteratorTask implements model.TaskBehavior
type IteratorTaskBehavior struct {
	TaskBehavior
}

// Eval implements model.TaskBehavior.Eval
func (tb *IteratorTaskBehavior) Eval(ctx model.TaskContext) (evalResult model.EvalResult, err error) {

	if ctx.Status() == model.TaskStatusSkipped {
		return model.EVAL_DONE, nil //todo introduce EVAL_SKIP?
	}

	task := ctx.Task()
	log.Debugf("Eval Iterator Task '%s'", task.ID())

	var itx Iterator

	itxAttr, ok := ctx.GetWorkingData("_iterator")
	iterationAttr, _ := ctx.GetWorkingData("iteration")

	if ok {
		itx = itxAttr.Value().(Iterator)
	} else {

		iterateOn, ok := getIterateValue(ctx)

		if !ok {
			//todo if iterateOn is not defined, what should we do?
			//just skip for now
			return model.EVAL_DONE, nil
		}

		switch t := iterateOn.(type) {
		case string:
			count, err := data.CoerceToInteger(iterateOn)
			if err != nil {
				return model.EVAL_FAIL, err
			}
			itx = NewIntIterator(count)
		case int:
			count := iterateOn.(int)
			itx = NewIntIterator(count)
		case map[string]interface{}:
			itx = NewObjectIterator(t)
		case []interface{}:
			itx = NewArrayIterator(t)
		default:
			return model.EVAL_FAIL, fmt.Errorf("unsupported type '%s' for iterateOn", t)
		}

		itxAttr, _ = data.NewAttribute("_iterator", data.ANY, itx)
		ctx.AddWorkingData(itxAttr)

		iteration := map[string]interface{}{
			"key":   nil,
			"value": nil,
		}

		iterationAttr, _ = data.NewAttribute("iteration", data.OBJECT, iteration)
		ctx.AddWorkingData(iterationAttr)
	}

	repeat := itx.next()

	if repeat {
		log.Debugf("Repeat:%s, Key:%s, Value:%v", repeat, itx.Key(), itx.Value())

		iteration, _ := iterationAttr.Value().(map[string]interface{})
		iteration["key"] = itx.Key()
		iteration["value"] = itx.Value()

		done, err := ctx.EvalActivity()

		if err != nil {
			log.Errorf("Error evaluating activity '%s'[%s] - %s", ctx.Task().Name(), ctx.Task().ActivityConfig().Ref(), err.Error())
			ctx.SetStatus(model.TaskStatusFailed)
			return model.EVAL_FAIL, err
		}

		if !done {
			ctx.SetStatus(model.TaskStatusWaiting)
			return model.EVAL_WAIT, nil
		}

		evalResult = model.EVAL_REPEAT

	} else {
		evalResult = model.EVAL_DONE
	}

	return evalResult, nil
}

// PostEval implements model.TaskBehavior.PostEval
func (tb *IteratorTaskBehavior) PostEval(ctx model.TaskContext) (evalResult model.EvalResult, err error) {

	log.Debugf("PostEval Iterator Task '%s'", ctx.Task().ID())

	_, err = ctx.PostEvalActivity()

	//what to do if eval isn't "done"?
	if err != nil {
		log.Errorf("Error post evaluating activity '%s'[%s] - %s", ctx.Task().Name(), ctx.Task().ActivityConfig().Ref(), err.Error())
		ctx.SetStatus(model.TaskStatusFailed)
		return model.EVAL_FAIL, err
	}

	itxAttr, _ := ctx.GetWorkingData("_iterator")
	itx := itxAttr.Value().(Iterator)

	if itx.HasNext() {
		return model.EVAL_REPEAT, nil
	}

	return model.EVAL_DONE, nil
}

func getIterateValue(ctx model.TaskContext) (value interface{}, set bool) {

	value, set = ctx.Task().GetSetting("iterate")
	if !set {
		return nil, false
	}

	strVal, ok := value.(string)
	if ok {
		if strVal[0] == '$' {
			val, err := ctx.Resolve(strVal)
			if err != nil {
				return val, true
			}
		}
	}

	return value, true
}

///////////////////////////////////
// Iterators

type Iterator interface {
	Key() interface{}
	Value() interface{}
	next() bool
	HasNext() bool
}

type ArrayIterator struct {
	current int
	data    []interface{}
}

func (itx *ArrayIterator) Key() interface{} {
	return itx.current
}

func (itx *ArrayIterator) Value() interface{} {
	return itx.data[itx.current]
}

func (itx *ArrayIterator) HasNext() bool {
	if itx.current >= len(itx.data) {
		return false
	}
	return true
}

func (itx *ArrayIterator) next() bool {
	itx.current++
	if itx.current >= len(itx.data) {
		return false
	}
	return true
}

func NewArrayIterator(data []interface{}) *ArrayIterator {
	return &ArrayIterator{data: data, current: -1}
}

type IntIterator struct {
	current int
	count   int
}

func (itx *IntIterator) Key() interface{} {
	return itx.current
}

func (itx *IntIterator) Value() interface{} {
	return itx.current
}

func (itx *IntIterator) HasNext() bool {
	if itx.current >= itx.count {
		return false
	}
	return true
}

func (itx *IntIterator) next() bool {
	itx.current++
	if itx.current >= itx.count {
		return false
	}
	return true
}

func NewIntIterator(count int) *IntIterator {
	return &IntIterator{count: count, current: -1}
}

type ObjectIterator struct {
	current int
	keyMap  map[int]string
	data    map[string]interface{}
}

func (itx *ObjectIterator) Key() interface{} {
	return itx.keyMap[itx.current]
}

func (itx *ObjectIterator) Value() interface{} {
	key := itx.keyMap[itx.current]
	return itx.data[key]
}

func (itx *ObjectIterator) HasNext() bool {
	if itx.current >= len(itx.data) {
		return false
	}
	return true
}

func (itx *ObjectIterator) next() bool {
	itx.current++
	if itx.current >= len(itx.data) {
		return false
	}
	return true
}

func NewObjectIterator(data map[string]interface{}) *ObjectIterator {
	keyMap := make(map[int]string, len(data))
	i := 0
	for key := range data {
		keyMap[i] = key
		i++
	}

	return &ObjectIterator{keyMap: keyMap, data: data, current: -1}
}

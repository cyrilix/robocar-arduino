package types

/* Radio control value */
type RCValue struct {
	Value      float64
	Confidence float64
}

type Steering RCValue
type Throttle RCValue

package tools

import (
	"reflect"
	"testing"
)

func TestNewThresholdConfigFromJson(t *testing.T) {
	type args struct {
		fileName string
	}
	tests := []struct {
		name    string
		args    args
		want    *ThresholdConfig
		wantErr bool
	}{
		{
			name: "default config",
			args: args{
				fileName: "test_data/config.json",
			},
			want: &defaultThresholdConfig,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewThresholdConfigFromJson(tt.args.fileName)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewThresholdConfigFromJson() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(*got, *tt.want) {
				t.Errorf("NewThresholdConfigFromJson() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got.MinValid, tt.want.MinValid) {
				t.Errorf("NewThresholdConfigFromJson(), bad minValid value: got = %v, want %v", got.MinValid, tt.want.MinValid)
			}
			if !reflect.DeepEqual(got.ThresholdSteps, tt.want.ThresholdSteps) {
				t.Errorf("NewThresholdConfigFromJson(), bad ThresholdSteps: got = %v, want %v", got.ThresholdSteps, tt.want.ThresholdSteps)
			}
		})
	}
}

func TestThresholdConfig_ValueOf(t *testing.T) {
	type fields struct {
		ThresholdSteps []float64
		MinValue       int
		Data           []int
	}
	type args struct {
		pwm int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   float64
	}{
		{
			name: "big value",
			fields: fields{
				ThresholdSteps: defaultThresholdConfig.ThresholdSteps,
				MinValue:       defaultThresholdConfig.MinValid,
				Data:           defaultThresholdConfig.Data,
			},
			args: args{
				pwm: 11000.,
			},
			want: 0,
		},
		{
			name: "little value",
			fields: fields{
				ThresholdSteps: defaultThresholdConfig.ThresholdSteps,
				MinValue:       defaultThresholdConfig.MinValid,
				Data:           defaultThresholdConfig.Data,
			},
			args: args{
				pwm: defaultThresholdConfig.MinValid - 1,
			},
			want: 0,
		},
		{
			name: "pwm at limit",
			fields: fields{
				ThresholdSteps: defaultThresholdConfig.ThresholdSteps,
				MinValue:       defaultThresholdConfig.MinValid,
				Data:           defaultThresholdConfig.Data,
			},
			args: args{
				pwm: defaultThresholdConfig.Data[2],
			},
			want: defaultThresholdConfig.ThresholdSteps[2],
		},
		{
			name: "between 2 limits",
			fields: fields{
				ThresholdSteps: defaultThresholdConfig.ThresholdSteps,
				MinValue:       defaultThresholdConfig.MinValid,
				Data:           defaultThresholdConfig.Data,
			},
			args: args{
				pwm: 800,
			},
			want: 0.275,
		},
		{
			name: "over last value and > minValue",
			fields: fields{
				ThresholdSteps: defaultThresholdConfig.ThresholdSteps,
				MinValue:       defaultThresholdConfig.MinValid,
				Data:           defaultThresholdConfig.Data,
			},
			args: args{
				pwm: defaultThresholdConfig.MinValid + 3,
			},
			want: 1.,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &ThresholdConfig{
				ThresholdSteps: tt.fields.ThresholdSteps,
				MinValid:       tt.fields.MinValue,
				Data:           tt.fields.Data,
			}
			got := f.ValueOf(tt.args.pwm)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ValueOf() = %v, want %v", got, tt.want)
			}
		})
	}
}

package main;

// type LogLine []rune

// type Template struct {
// 	Mask []rune // Y Y [X] Y
// 	Labels []string // Date Time Message Log_Level 
// }

// type Parser interface {
// 	Parse(LogLine) map[string][]rune
// }

// type TemplateParser struct {
// 	Template Template
// }

// func NewTemplateParser(template Template) *TemplateParser {
// 	return &TemplateParser{
// 		Template: template,
// 	}
// }

// func (p *TemplateParser) Parse(input LogLine) map[string][]rune {
// 	mask := p.Template.Mask
// 	labels := p.Template.Labels

// 	output := make(map[string][]rune)

// 	for _, label := range labels {
// 		output[label] = []rune{}
// 	}

// 	labelCount := 0
// 	maskWalk := 0
// 	logWalk := 0
// 	for maskWalk < len(mask) {
// 		if mask[maskWalk] == 'Y' {
// 			for logWalk < len(input) {
// 				r := input[logWalk]
// 				if !(r >= 'a' && r <= 'z') && !(r >= 'A' && r <= 'Z') && !(r >= '0' && r <= '9') {
// 					break
// 				}

// 				output[labels[labelCount]] = append(output[labels[labelCount]], r)
// 				logWalk++
// 			}
// 		} else if mask[maskWalk] == 'X' {

// 		}

// 		maskWalk++
// 		labelCount++
// 		logWalk++
// 	}


// 	return nil
// }
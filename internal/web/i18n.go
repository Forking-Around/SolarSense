package web

var messages = map[string]map[string]string{
	"en": {"title": "Does solar make sense for you?", "subtitle": "A clear estimate using your roof, sunlight, usage and local solar data.", "location": "Where is the installation?", "bill": "Electricity and usage", "roof": "Your roof", "sun": "When does the roof get direct sun?", "backup": "Power cuts and backup", "calculate": "Calculate my solar fit"},
	"te": {"title": "మీకు సోలార్ సరిపోతుందా?", "subtitle": "మీ పైకప్పు, సూర్యకాంతి, వినియోగం మరియు స్థానిక సోలార్ డేటాతో స్పష్టమైన అంచనా.", "location": "సోలార్ ఎక్కడ ఏర్పాటు చేయాలి?", "bill": "విద్యుత్ బిల్లు మరియు వినియోగం", "roof": "మీ పైకప్పు", "sun": "పైకప్పుపై నేరుగా ఎప్పుడు సూర్యకాంతి పడుతుంది?", "backup": "విద్యుత్ కోతలు మరియు బ్యాకప్", "calculate": "నా సోలార్ అంచనా చూపించు"},
	"hi": {"title": "क्या सोलर आपके लिए सही है?", "subtitle": "आपकी छत, धूप, उपयोग और स्थानीय सोलर डेटा पर आधारित स्पष्ट अनुमान।", "location": "सोलर कहाँ लगाना है?", "bill": "बिजली बिल और उपयोग", "roof": "आपकी छत", "sun": "छत पर सीधी धूप कब आती है?", "backup": "बिजली कटौती और बैकअप", "calculate": "मेरा सोलर अनुमान दिखाएँ"},
}

func language(code string) map[string]string {
	if m, ok := messages[code]; ok {
		return m
	}
	return messages["en"]
}

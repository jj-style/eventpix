// Static data for template rendering
package templatedata

type Features []*Feature

type Feature struct {
	Icon         string        `json:"icon"`
	Title        string        `json:"title"`
	Description  string        `json:"description"`
	CallToAction *CallToAction `json:"callToAction"`
}

type CallToAction struct {
	Href string `json:"href"`
	Text string `json:"text"`
}

type PriceTiers struct {
	Title    string       `json:"title"`
	Subtitle string       `json:"subtitle"`
	Tiers    []*PriceTier `json:"tiers"`
}

type PriceTier struct {
	Title    string         `json:"title"`
	Price    string         `json:"price"`
	PricePer string         `json:"pricePer"`
	Includes []*TierInclude `json:"includes"`
}

type TierInclude struct {
	Name     string `json:"name"`
	Included bool   `json:"included"`
	Bold     bool   `json:"bold"`
}

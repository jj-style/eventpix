// Static data for template rendering
package templatedata

var (
	IndexFeatures Features = Features{
		&Feature{
			Icon:        "bi-device-hdd",
			Title:       "Bring Your Own Storage",
			Description: "All uploaded media is stored directly in your storage, giving you full ownership of the media with no storage limitations.",
		},
		&Feature{
			Icon:        "bi-code",
			Title:       "Open Source",
			Description: "Source code is open-source and fully self-hostable, building trust, inviting collaboration and allowing you to run your own server.",
			CallToAction: &CallToAction{
				Href: "https://github.com/jj-style",
				Text: "Source code repository",
			},
		},
		&Feature{
			Icon:        "bi-qr-code-scan",
			Title:       "QR Code & Simple UI",
			Description: "Intuitive easy to use interface, with QR codes for your event to share with your guests.",
		},
	}

	IndexPriceTiers PriceTiers = PriceTiers{
		Title: "Tiers",
		Tiers: []*PriceTier{
			{
				Title: "free",
				Price: "£0",
				Includes: []*TierInclude{
					{
						Name:     "3 events",
						Included: true,
					},
					{
						Name:     "QR code for event",
						Included: true,
					},
					{
						Name:     "24/7 support",
						Included: false,
					},
					{
						Name:     "your own server running either remotely or at your event",
						Included: false,
					},
					{
						Name:     "custom domain",
						Included: false,
					},
				},
			},
			/*
				{
					Title: "pro",
					Price: "£25",
					Includes: []*TierInclude{
						{
							Name:     "unlimited events",
							Included: true,
							Bold:     true,
						},
						{
							Name:     "QR code for event",
							Included: true,
						},
						{
							Name:     "24/7 support",
							Included: true,
						},
						{
							Name:     "your own server running either remotely or at your event",
							Included: false,
						},
						{
							Name:     "custom domain",
							Included: false,
						},
					},
				},
			*/
			{
				Title:    "self-hosted instance",
				Price:    "POA",
				PricePer: " (enquire below) ",
				Includes: []*TierInclude{
					{
						Name:     "unlimited events",
						Included: true,
						Bold:     true,
					},
					{
						Name:     "QR code for event",
						Included: true,
					},
					{
						Name:     "24/7 support",
						Included: true,
					},
					{
						Name:     "your own server running either remotely or at your event",
						Included: true,
					},
					{
						Name:     "custom domain",
						Included: true,
					},
				},
			},
		},
	}
)

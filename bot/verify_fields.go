package bot

import (
	"github.com/bwmarrin/discordgo"

	"license-bot-go/scrapers"
)

func buildLicenseFields(match *scrapers.LicenseResult, state string) []*discordgo.MessageEmbedField {
	fields := []*discordgo.MessageEmbedField{
		{Name: "Full Name", Value: nvl(match.FullName, "N/A"), Inline: true},
		{Name: "State", Value: state, Inline: true},
		{Name: "Status", Value: nvl(match.Status, "N/A"), Inline: true},
		{Name: "License #", Value: nvl(match.LicenseNumber, "N/A"), Inline: true},
		{Name: "NPN", Value: nvl(match.NPN, "N/A"), Inline: true},
		{Name: "License Type", Value: nvl(match.LicenseType, "N/A"), Inline: true},
	}

	if match.IssueDate != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name: "Issue Date", Value: match.IssueDate, Inline: true,
		})
	}
	if match.ExpirationDate != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name: "Expiration Date", Value: match.ExpirationDate, Inline: true,
		})
	}
	if match.Resident {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name: "Residency", Value: "Resident", Inline: true,
		})
	}
	if match.LOAs != "" {
		loas := match.LOAs
		if len(loas) > 900 {
			loas = loas[:900] + "..."
		}
		fields = append(fields, &discordgo.MessageEmbedField{
			Name: "Lines of Authority", Value: loas, Inline: false,
		})
	}
	if match.BusinessAddress != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name: "Business Address", Value: match.BusinessAddress, Inline: false,
		})
	}
	if match.BusinessPhone != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name: "Business Phone", Value: match.BusinessPhone, Inline: true,
		})
	}
	if match.Email != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name: "Email", Value: match.Email, Inline: true,
		})
	}
	if match.County != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name: "County", Value: match.County, Inline: true,
		})
	}

	fields = append(fields, &discordgo.MessageEmbedField{
		Name: "\u200b\nNext Step: Contracting",
		Value: "Use `/contract` in the server to book your contracting appointment.\n\n" +
			"**What to Prepare:**\n" +
			"- Government-issued photo ID\n" +
			"- Social Security number\n" +
			"- E&O insurance info\n" +
			"- Bank info for direct deposit\n" +
			"- Resident state license number",
		Inline: false,
	})

	return fields
}

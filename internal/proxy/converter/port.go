package converter

import (
	"twist/internal/api"
	"twist/internal/proxy/database"
)

// ConvertTPortToPortInfo converts database TPort to API PortInfo
func ConvertTPortToPortInfo(sectorID int, tport database.TPort) (*api.PortInfo, error) {
	
	// Skip conversion if port has no class (no port present)
	if tport.ClassIndex == 0 {
		return nil, nil
	}
	
	// Convert class index to PortClass enum
	classType := api.PortClass(tport.ClassIndex)
	if tport.ClassIndex == 9 {
		classType = api.PortClassSTD // Special case for Stardock
	}
	
	// Convert products
	var products []api.ProductInfo
	for i := 0; i < 3; i++ {
		productType := api.ProductType(i) // FuelOre=0, Organics=1, Equipment=2
		
		var status api.ProductStatus
		if tport.BuyProduct[i] {
			status = api.ProductStatusBuying
		} else if tport.ProductAmount[i] > 0 {
			status = api.ProductStatusSelling
		} else {
			status = api.ProductStatusNone
		}
		
		products = append(products, api.ProductInfo{
			Type:       productType,
			Status:     status,
			Quantity:   tport.ProductAmount[i],
			Percentage: tport.ProductPercent[i],
		})
	}
	
	portInfo := &api.PortInfo{
		SectorID:   sectorID,
		Name:       tport.Name,
		Class:      tport.ClassIndex,
		ClassType:  classType,
		BuildTime:  tport.BuildTime,
		Products:   products,
		LastUpdate: tport.UpDate,
		Dead:       tport.Dead,
	}
	
	return portInfo, nil
}

// ConvertPortClassToString converts port class index to port type string (legacy support)
func ConvertPortClassToString(classIndex int) string {
	switch classIndex {
	case 1:
		return "BBS"
	case 2:
		return "BSB"
	case 3:
		return "SBB"
	case 4:
		return "SSB"
	case 5:
		return "SBS"
	case 6:
		return "BSS"
	case 7:
		return "SSS"
	case 8:
		return "BBB"
	case 9:
		return "STD" // Special case for stardock/federation ports
	default:
		return ""    // No port or unknown class
	}
}
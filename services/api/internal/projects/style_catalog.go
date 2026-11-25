package projects

type houseStyleDefinition struct {
	Name  string
	Image string
}

var houseStyleCatalog = map[string]houseStyleDefinition{
	"modern-minimalist":               {"Modern Minimalist", "/assets/house-types/01_Rumah_Minimalis_Modern.jpg"},
	"tropical-classic":                {"Tropical Classic", "/assets/house-types/02_Rumah_Tropis_Klasik.jpg"},
	"scandinavian":                    {"Scandinavian", "/assets/house-types/03_Rumah_Scandinavian.jpg"},
	"industrial":                      {"Industrial", "/assets/house-types/04_Rumah_Industrial.jpg"},
	"industrial-grey":                 {"Industrial Grey", "/assets/house-types/05_Rumah_Industrial_Variasi_Abu_Abu.jpg"},
	"japanese-minka":                  {"Japanese Traditional Minka", "/assets/house-types/06_Rumah_Jepang_Tradisional_Minka.jpg"},
	"contemporary":                    {"Contemporary", "/assets/house-types/07_Rumah_Kontemporer.jpg"},
	"mediterranean":                   {"Mediterranean", "/assets/house-types/08_Rumah_Mediterranean.jpg"},
	"colonial":                        {"Colonial", "/assets/house-types/09_Rumah_Colonial.jpg"},
	"colonial-symmetrical":            {"Colonial Symmetrical", "/assets/house-types/10_Rumah_Colonial_Variasi_Simetris.jpg"},
	"rustic-wood":                     {"Rustic Wood", "/assets/house-types/11_Rumah_Rustic_Kayu.jpg"},
	"modern-tropical":                 {"Modern Tropical", "/assets/house-types/12_Rumah_Modern_Tropis.jpg"},
	"victorian":                       {"Victorian", "/assets/house-types/13_Rumah_Victorian.jpg"},
	"futuristic":                      {"Futuristic", "/assets/house-types/14_Rumah_Futuristik.jpg"},
	"futuristic-capsule":              {"Futuristic Capsule", "/assets/house-types/15_Rumah_Futuristik_Variasi_Kapsul.jpg"},
	"english-cottage":                 {"English Cottage", "/assets/house-types/16_Rumah_Cottage_Inggris.jpg"},
	"modern-black":                    {"Modern Black", "/assets/house-types/17_Rumah_Modern_Hitam.jpg"},
	"traditional-stilt-house":         {"Traditional Stilt House", "/assets/house-types/18_Rumah_Panggung_Tradisional.jpg"},
	"traditional-stilt-house-variant": {"Traditional Stilt House Variant", "/assets/house-types/19_Rumah_Panggung_Tradisional_Variasi.jpg"},
	"bauhaus":                         {"Bauhaus", "/assets/house-types/20_Rumah_Bauhaus.jpg"},
	"luxury-villa":                    {"Luxury Villa", "/assets/house-types/21_Rumah_Villa_Mewah.jpg"},
	"tiny-house":                      {"Tiny House", "/assets/house-types/22_Rumah_Tiny_House.jpg"},
	"art-deco":                        {"Art Deco", "/assets/house-types/23_Rumah_Art_Deco.jpg"},
	"art-deco-geometric":              {"Art Deco Geometric", "/assets/house-types/24_Rumah_Art_Deco_Variasi_Geometris.jpg"},
	"ecological-green-house":          {"Ecological Green House", "/assets/house-types/25_Rumah_Ekologis_Green_House.jpg"},
}

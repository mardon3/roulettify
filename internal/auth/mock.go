package auth

import (
	"fmt"
	"math/rand"
)

// Mock track data for testing
var mockTrackPool = []struct {
	Name    string
	Artists []string
}{
	{"Blinding Lights", []string{"The Weeknd"}},
	{"Shape of You", []string{"Ed Sheeran"}},
	{"Someone Like You", []string{"Adele"}},
	{"Uptown Funk", []string{"Mark Ronson", "Bruno Mars"}},
	{"Thinking Out Loud", []string{"Ed Sheeran"}},
	{"Levitating", []string{"Dua Lipa"}},
	{"drivers license", []string{"Olivia Rodrigo"}},
	{"Shallow", []string{"Lady Gaga", "Bradley Cooper"}},
	{"Watermelon Sugar", []string{"Harry Styles"}},
	{"Bad Guy", []string{"Billie Eilish"}},
	{"Dance Monkey", []string{"Tones and I"}},
	{"Circles", []string{"Post Malone"}},
	{"Memories", []string{"Maroon 5"}},
	{"Señorita", []string{"Shawn Mendes", "Camila Cabello"}},
	{"Old Town Road", []string{"Lil Nas X", "Billy Ray Cyrus"}},
	{"Sunflower", []string{"Post Malone", "Swae Lee"}},
	{"Perfect", []string{"Ed Sheeran"}},
	{"Havana", []string{"Camila Cabello", "Young Thug"}},
	{"Closer", []string{"The Chainsmokers", "Halsey"}},
	{"Despacito", []string{"Luis Fonsi", "Daddy Yankee"}},
	{"Stay", []string{"The Kid LAROI", "Justin Bieber"}},
	{"Good 4 U", []string{"Olivia Rodrigo"}},
	{"Heat Waves", []string{"Glass Animals"}},
	{"Save Your Tears", []string{"The Weeknd"}},
	{"Peaches", []string{"Justin Bieber", "Daniel Caesar"}},
	{"Montero", []string{"Lil Nas X"}},
	{"Industry Baby", []string{"Lil Nas X", "Jack Harlow"}},
	{"Levitating", []string{"Dua Lipa", "DaBaby"}},
	{"Positions", []string{"Ariana Grande"}},
	{"Mood", []string{"24kGoldn", "iann dior"}},
	{"Therefore I Am", []string{"Billie Eilish"}},
	{"Dynamite", []string{"BTS"}},
	{"Butter", []string{"BTS"}},
	{"Permission to Dance", []string{"BTS"}},
	{"Easy On Me", []string{"Adele"}},
	{"Shivers", []string{"Ed Sheeran"}},
	{"Cold Heart", []string{"Elton John", "Dua Lipa"}},
	{"Essence", []string{"Wizkid", "Tems"}},
	{"Fancy Like", []string{"Walker Hayes"}},
	{"My Universe", []string{"Coldplay", "BTS"}},
	{"Beggin", []string{"Måneskin"}},
	{"Happier Than Ever", []string{"Billie Eilish"}},
	{"Kiss Me More", []string{"Doja Cat", "SZA"}},
	{"Woman", []string{"Doja Cat"}},
	{"Need to Know", []string{"Doja Cat"}},
	{"Levitating", []string{"Dua Lipa"}},
	{"Take My Breath", []string{"The Weeknd"}},
	{"Bad Habits", []string{"Ed Sheeran"}},
	{"Stay With Me", []string{"Sam Smith"}},
	{"Love Yourself", []string{"Justin Bieber"}},
	{"Sorry", []string{"Justin Bieber"}},
}

// GenerateMockPlayer creates a mock player with fake Spotify data
func GenerateMockPlayer(index int) *Player {
	guestNames := []string{
		"Alex", "Jordan", "Taylor", "Morgan", "Casey",
		"Riley", "Avery", "Quinn", "Skylar", "Drew",
	}
	
	name := guestNames[index%len(guestNames)]
	if index >= len(guestNames) {
		name = fmt.Sprintf("%s%d", name, index/len(guestNames)+1)
	}
	
	playerID := fmt.Sprintf("guest_%d", index)
	
	return &Player{
		ID:          playerID,
		Name:        name,
		SpotifyID:   playerID,
		AccessToken: "mock_token_" + playerID,
		TopTracks:   generateMockTopTracks(index),
	}
}

// generateMockTopTracks generates 50 random tracks with varied rankings
func generateMockTopTracks(seed int) []Track {
	// Use seed to make each player's tracks slightly different
	r := rand.New(rand.NewSource(int64(seed * 12345)))
	
	// Shuffle track pool for this player
	shuffled := make([]struct {
		Name    string
		Artists []string
	}, len(mockTrackPool))
	copy(shuffled, mockTrackPool)
	
	r.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	
	tracks := make([]Track, 50)
	for i := 0; i < 50; i++ {
		trackData := shuffled[i%len(shuffled)]
		
		tracks[i] = Track{
			ID:         fmt.Sprintf("mock_track_%d_%d", seed, i),
			Name:       trackData.Name,
			Artists:    trackData.Artists,
			Rank:       i + 1,
			URI:        fmt.Sprintf("spotify:track:mock_%d_%d", seed, i),
			ImageURL:   "https://via.placeholder.com/300x300?text=Album+Art",
			PreviewURL: "", // No preview for mock data
		}
	}
	
	return tracks
}

// IsMockPlayer checks if a player is using mock data
func IsMockPlayer(playerID string) bool {
	return len(playerID) > 6 && playerID[:6] == "guest_"
}
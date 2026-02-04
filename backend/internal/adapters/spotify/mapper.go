package spotify

import "github.com/ewilliams-labs/overture/backend/internal/core/domain"

// mapTrackToDomain converts a spotifyTrack to a domain.Track.
func mapTrackToDomain(st spotifyTrack) domain.Track {
    return domain.Track{
        ID:     st.ID,
        Title:  st.Title,
        Artist: st.Artist,
        Album:  st.Album,
        ISRC:   st.ISRC,
        Vibe:   st.Vibe,
    }
}

// mapPlaylistToDomain converts a spotifyPlaylist to a domain.Playlist.
func mapPlaylistToDomain(sp spotifyPlaylist) domain.Playlist {
    tracks := make([]domain.Track, len(sp.Tracks))
    for i, st := range sp.Tracks {
        tracks[i] = mapTrackToDomain(st)
    }
    return domain.Playlist{
        ID:     sp.ID,
        Name:   sp.Name,
        Tracks: tracks,
    }
}
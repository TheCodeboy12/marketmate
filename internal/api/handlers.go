package api

import "net/http"

func (api *API) MeHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
	}
	if me, err := api.store.GetUserById(r.Context(), user.UserID); err != nil {
		writeError(w, http.StatusInternalServerError, "Unable to find you")
		return
	} else {
		writeJSON(w, http.StatusOK, me)
	}
}

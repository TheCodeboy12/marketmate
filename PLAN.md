# PLAN for this shopping list app

## api

The api methods we will expose are:

| endpoint | http method | description |
| -------------- | --------------- | --------------- |
| /login | POST | The user will provide: `email` and `password` in the body of the request. This endpoint is not protected by the JWT |

### Authentication and auth

we are going to use jwt autherazation using claims that are :

```go
type Claims struct {
 UserID   int64 `json:"user_id"` TenantID int64 `json:"tenant_id"`
 RoleID   int64 `json:"role_id"`
}
```

When the user signs in, we validate with the db that their provided password hash matches our hash.
If they match the API will issue a JWT using the `GenerateToken` function in the `internal/auth` package.

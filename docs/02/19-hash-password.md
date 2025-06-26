# Hash password

<https://github.com/go-playground/validator> 裡面有很墮有用的內建 tag ，像是 email

```go
type CreateUserRequest struct {
    Username string `json:"username" binding:"required,alphanum"`
    Password string `json:"password" binding:"required,min=6"`
    FullName string `json:"full_name" binding:"required"`
    Email    string `json:"email" binding:"required,email"`
}
```

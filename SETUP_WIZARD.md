# First-Launch Setup Wizard

## Overview
Implemented a first-launch setup wizard that automatically detects when the database is empty and guides the user through creating the first administrator account.

## How It Works

### 1. Detection
- Middleware checks `db.IsFirstUser()` on every page load
- If no users exist, redirects to `/setup`
- Bypasses check for setup page and registration API

### 2. Setup Page (`/setup`)
**Features:**
- Clean, welcoming interface
- Form fields:
  - Username (validated: lowercase, numbers, hyphens)
  - Email (validated)
  - Password (minimum 8 characters)
  - Confirm Password
- Real-time validation
- Loading state during creation
- Error handling

**User Experience:**
- Clear messaging: "This is the first time shebang.run is being launched"
- Explains admin privileges
- Password strength requirements
- Confirmation before submission

### 3. Admin Creation
- Calls existing `/api/auth/register` endpoint
- First user automatically gets `is_admin = true`
- Stores JWT token in localStorage
- Stores user info in localStorage

### 4. Redirect
- After successful creation, redirects to `/dashboard`
- User is immediately logged in
- Can start using the platform

## API Endpoints

### `GET /api/setup/status`
Returns setup status:
```json
{
  "setup_complete": false,
  "needs_setup": true
}
```

Used by setup page to verify setup is still needed.

## Middleware Logic

```go
setupCheck := func(next http.Handler) http.Handler {
    // Check if first user
    isFirst, err := db.IsFirstUser()
    if err == nil && isFirst {
        // Redirect to setup
        http.Redirect(w, r, "/setup", http.StatusTemporaryRedirect)
        return
    }
    next.ServeHTTP(w, r)
}
```

**Bypassed for:**
- `/setup` - The setup page itself
- `/api/setup/status` - Status check endpoint
- `/api/auth/register` - Registration endpoint

## User Flow

```
1. User visits http://localhost
   ↓
2. Middleware detects no users exist
   ↓
3. Redirects to /setup
   ↓
4. User fills in admin credentials
   ↓
5. Submits form → POST /api/auth/register
   ↓
6. First user created with is_admin=true
   ↓
7. JWT token stored in localStorage
   ↓
8. Redirects to /dashboard
   ↓
9. User is logged in as admin
```

## Security Considerations

✅ **Password Requirements:**
- Minimum 8 characters
- Must match confirmation
- Hashed with bcrypt before storage

✅ **Validation:**
- Username pattern: `[a-z0-9-]+`
- Email format validation
- Client and server-side validation

✅ **Protection:**
- Setup page only accessible when needed
- After first user created, setup redirects to home
- No way to bypass or create multiple "first" admins

## Files Modified/Created

**New Files:**
- `web/templates/setup.html` - Setup wizard UI
- `internal/api/setup.go` - Setup status API

**Modified Files:**
- `internal/api/web.go` - Added setup handler
- `cmd/server/main.go` - Added middleware and routes

## Testing

**First Launch:**
1. Start fresh database
2. Visit http://localhost
3. Should redirect to /setup
4. Create admin account
5. Should redirect to /dashboard
6. Verify admin privileges

**After Setup:**
1. Visit http://localhost
2. Should show normal homepage
3. /setup should redirect to home
4. Can register additional users (non-admin)

## Benefits

✅ **User-Friendly:**
- No need to know about registration
- Clear instructions
- Guided process

✅ **Secure:**
- Forces admin creation on first launch
- Prevents unauthorized access
- Proper validation

✅ **Professional:**
- Clean UI matching site design
- Proper error handling
- Loading states

✅ **Automatic:**
- No manual configuration needed
- Detects empty database
- Self-configuring

## Example

**First Visit:**
```
┌─────────────────────────────────────┐
│   Welcome to shebang.run            │
│   Let's set up your admin account   │
│                                     │
│   🎉 First time launch detected     │
│                                     │
│   Username: [admin________]         │
│   Email:    [admin@example.com]     │
│   Password: [••••••••]              │
│   Confirm:  [••••••••]              │
│                                     │
│   [Create Administrator Account]    │
└─────────────────────────────────────┘
```

**After Setup:**
- Normal homepage shows
- Login/Register available
- Setup page redirects away

## Conclusion

The first-launch setup wizard provides a professional, user-friendly onboarding experience that:
1. ✅ Detects empty database
2. ✅ Shows setup wizard
3. ✅ Creates first admin user
4. ✅ Redirects to dashboard

This ensures every deployment starts with a proper administrator account without requiring manual intervention or documentation reading.

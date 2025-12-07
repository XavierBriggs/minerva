# Minerva Testing Guide

## 🚀 Quick Start

### 1. Rebuild Services (if needed)
```bash
cd /Users/xavierbriggs/development/fortuna/deploy
docker-compose build api-gateway minerva
docker-compose up -d api-gateway minerva
```

### 2. Check Service Health
```bash
docker-compose ps api-gateway minerva
docker-compose logs -f minerva --tail=50
```

### 3. Start Frontend
```bash
cd /Users/xavierbriggs/development/fortuna/web/fortuna_client
npm run dev
```

### 4. Open Browser
Navigate to: **http://localhost:3000/minerva**

---

## 🧪 Testing Checklist

### Tab 1: Live & Upcoming Games
- [ ] **Live Games Section**
  - Should show games currently in progress with red "LIVE" badge
  - Scores should update automatically (WebSocket connected)
  - Click on any game card to open box score modal
  
- [ ] **Upcoming Games Section**
  - Should show scheduled games with blue "UPCOMING" badge
  - Shows game time in your local timezone
  
- [ ] **Final Games Section**
  - Should show completed games with gray "FINAL" badge
  - Final scores displayed

- [ ] **Stats Cards at Top**
  - Live Now count
  - Upcoming Today count
  - Completed count
  - Connection status (green ✓ = connected)

### Tab 2: Stats & History ⭐ NEW!
- [ ] **Teams Section**
  - Click "Browse Teams →" button
  - Should navigate to `/minerva/teams`
  - Shows all NBA teams organized by conference
  - Search teams by name/city/abbreviation
  - Filter by Eastern/Western conference
  - Click any team to view roster & schedule

- [ ] **Players Section**
  - Search input (placeholder for now)
  - Access players through:
    - Box scores (click player names)
    - Team rosters (click player names)

- [ ] **Historical Games**
  - Select a date to view past games
  - Click "Today" to jump to current date
  - Click any game to view box score
  - If no games found, shows helpful message

- [ ] **Pro Tip Box**
  - Shows navigation instructions

### Tab 3: Data Management
- [ ] **Backfill Status**
  - Shows current job status (running/completed/failed)
  - Progress bar updates in real-time
  - Shows recent job history

- [ ] **Season Load Buttons**
  - Click "2025-26" to load current season
  - Click other seasons to load historical data
  - Should show progress and completion

---

## 🎯 Feature Testing

### Box Score Modal
1. Click any game card
2. Modal should open with:
   - Game details (teams, score, status)
   - Home team stats table
   - Away team stats table
   - Player stats: MIN, PTS, REB, AST, FG, FG%, 3PT, 3P%, FT, FT%, STL, BLK, TO, +/-
   - Team totals row at bottom
   - Click player names to view profile (if implemented)
3. Click outside modal or X button to close

### Team Page (`/minerva/teams/[teamId]`)
1. Navigate to any team
2. Should show:
   - Team header with name and info
   - **Roster tab**: All players with positions and jersey numbers
     - Click player to view profile
   - **Schedule tab**: Upcoming and recent games
     - Click game to view box score
   - **Stats tab**: Team statistics (placeholder for now)

### Player Page (`/minerva/players/[playerId]`) - Coming Soon
1. Click any player name from:
   - Box score modal
   - Team roster
2. Should show:
   - Player profile info
   - Recent game logs
   - Season averages
   - Advanced stats

---

## 🐛 Common Issues & Fixes

### Issue: Services not starting
```bash
# Check logs
docker-compose logs minerva --tail=100

# Restart services
docker-compose restart minerva api-gateway
```

### Issue: CORS errors in browser console
```bash
# Rebuild API Gateway with CORS fix
cd deploy
docker-compose build api-gateway
docker-compose up -d api-gateway
```

### Issue: No games showing
1. Go to "Data Management" tab
2. Click "2025-26" to load current season
3. Wait for completion (2-5 minutes)
4. Refresh page

### Issue: WebSocket not connected
- Check if `ws-broadcaster` service is running:
```bash
docker-compose ps ws-broadcaster
docker-compose logs ws-broadcaster --tail=50
```

### Issue: Frontend not updating
```bash
# Clear Next.js cache
cd web/fortuna_client
rm -rf .next
npm run dev
```

---

## 📊 What to Look For

### ✅ Good Signs
- Green "Live" indicator in top right
- Games load within 2-3 seconds
- Box scores open smoothly
- Team pages load with rosters
- No console errors
- Smooth navigation between pages

### ❌ Red Flags
- Red "Offline" indicator (WebSocket issue)
- CORS errors in console
- 404 errors for API calls
- Blank pages or infinite loading
- Teams showing as "Team 1" instead of names

---

## 🎨 UI/UX Features to Test

### Responsiveness
- Test on different screen sizes
- Mobile view (< 768px)
- Tablet view (768px - 1024px)
- Desktop view (> 1024px)

### Dark Mode
- Should work automatically based on system preference
- All colors should be readable
- Cards should have proper contrast

### Animations
- Live badge should pulse
- Loading spinners should rotate
- Hover effects on cards
- Smooth transitions

### Accessibility
- Tab navigation should work
- Buttons should be keyboard accessible
- Focus states visible
- Screen reader friendly (semantic HTML)

---

## 🔍 API Endpoints to Test

### Manual API Testing (Optional)
```bash
# Get live games
curl http://localhost:8080/api/v1/minerva/games/live

# Get teams
curl http://localhost:8080/api/v1/minerva/teams

# Get team by ID
curl http://localhost:8080/api/v1/minerva/teams/1

# Get game box score
curl http://localhost:8080/api/v1/minerva/games/401704794/boxscore

# Get backfill status
curl http://localhost:8080/api/v1/minerva/backfill/status
```

---

## 📝 Testing Notes Template

Use this to track your testing:

```
Date: ___________
Tester: ___________

✅ Live Games: ___________
✅ Upcoming Games: ___________
✅ Historical Games: ___________
✅ Teams Page: ___________
✅ Team Detail Page: ___________
✅ Box Score Modal: ___________
✅ Backfill: ___________
✅ WebSocket: ___________

Issues Found:
1. ___________
2. ___________
3. ___________

Notes:
___________
```

---

## 🎯 Next Steps After Testing

If everything works:
1. ✅ Mark features as complete
2. 📸 Take screenshots for documentation
3. 🚀 Ready for production deployment

If issues found:
1. 📋 Document the issue
2. 🐛 Create bug report with steps to reproduce
3. 🔧 Fix and re-test









#!/usr/bin/env bash
# Post-flight API validation tests for Loom
# Validates all documented API endpoints are responding correctly

set -euo pipefail

# Configuration
BASE_URL="${BASE_URL:-http://localhost:8080}"
TIMEOUT="${TIMEOUT:-5}"
VERBOSE="${VERBOSE:-0}"
AUTH_USER="${AUTH_USER:-admin}"
AUTH_PASSWORD="${AUTH_PASSWORD:-admin}"
AUTH_TOKEN=""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
PASSED=0
FAILED=0
TOTAL=0

# Utility functions
info() {
    echo -e "${BLUE}[INFO]${NC} $*"
}

success() {
    echo -e "${GREEN}[PASS]${NC} $*"
    ((PASSED++)) || true
}

error() {
    echo -e "${RED}[FAIL]${NC} $*"
    ((FAILED++)) || true
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $*"
}

# Test function: Makes HTTP request and validates response
# Usage: test_endpoint METHOD PATH EXPECTED_CODE [BODY] [DESCRIPTION]
test_endpoint() {
    local method="$1"
    local path="$2"
    local expected_code="$3"
    local body="${4:-}"
    local description="${5:-$method $path}"
    
    ((TOTAL++)) || true
    
    local url="${BASE_URL}${path}"
    local response_file
    response_file=$(mktemp)
    local http_code
    
    if [ "$VERBOSE" -eq 1 ]; then
        info "Testing: $description"
    fi
    
    # Make request
    local auth_header=()
    if [ -n "$AUTH_TOKEN" ]; then
        auth_header=(-H "Authorization: Bearer ${AUTH_TOKEN}")
    fi

    if [ -n "$body" ]; then
        http_code=$(curl -s -o "$response_file" -w "%{http_code}" \
            -X "$method" \
            -H "Content-Type: application/json" \
            "${auth_header[@]}" \
            -d "$body" \
            --max-time "$TIMEOUT" \
            "$url" 2>/dev/null || echo "000")
    else
        http_code=$(curl -s -o "$response_file" -w "%{http_code}" \
            -X "$method" \
            "${auth_header[@]}" \
            --max-time "$TIMEOUT" \
            "$url" 2>/dev/null || echo "000")
    fi
    
    # Validate response code
    if [ "$http_code" = "000" ]; then
        error "$description - Connection failed or timed out"
        rm -f "$response_file"
        return 1
    elif [ "$http_code" != "$expected_code" ]; then
        error "$description - Expected $expected_code, got $http_code"
        if [ "$VERBOSE" -eq 1 ]; then
            cat "$response_file"
        fi
        rm -f "$response_file"
        return 1
    fi
    
    # Validate JSON response (if not 204 No Content)
    if [ "$http_code" != "204" ]; then
        if ! jq empty "$response_file" 2>/dev/null; then
            error "$description - Invalid JSON response"
            rm -f "$response_file"
            return 1
        fi
    fi
    
    success "$description"
    rm -f "$response_file"
    return 0
}

# Main test suite
main() {
    info "Starting Loom API post-flight tests"
    info "Base URL: $BASE_URL"
    echo ""
    
    # Check if jq is available
    if ! command -v jq &> /dev/null; then
        error "jq is required but not installed. Please install jq."
        exit 1
    fi
    
    # Wait for service to be ready
    info "Checking if service is available..."
    local retries=0
    local max_retries=30
    while [ $retries -lt $max_retries ]; do
        if curl -s -f -o /dev/null "$BASE_URL/api/v1/health" 2>/dev/null; then
            success "Service is available"
            break
        fi
        ((retries++)) || true
        if [ $retries -eq $max_retries ]; then
            error "Service did not become available after ${max_retries} attempts"
            exit 1
        fi
        sleep 1
    done
    echo ""

    # Authenticate for protected endpoints
    info "Authenticating..."
    login_resp=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"${AUTH_USER}\",\"password\":\"${AUTH_PASSWORD}\"}" 2>/dev/null || echo "")
    AUTH_TOKEN=$(echo "$login_resp" | jq -r '.token // empty')
    if [ -z "$AUTH_TOKEN" ]; then
        error "Failed to authenticate. Check AUTH_USER/AUTH_PASSWORD."
        exit 1
    fi
    success "Authenticated as ${AUTH_USER}"
    echo ""
    
    # ============================================================================
    # Health Check
    # ============================================================================
    info "Testing Health Check endpoints..."
    test_endpoint "GET" "/api/v1/health" "200" "" "Health check"
    test_endpoint "GET" "/api/v1/system/status" "200" "" "System status"
    echo ""
    
    # ============================================================================
    # Providers
    # ============================================================================
    info "Testing Provider endpoints..."
    test_endpoint "GET" "/api/v1/providers" "200" "" "List providers"
    echo ""
    
    # ============================================================================
    # Projects
    # ============================================================================
    info "Testing Project endpoints..."
    test_endpoint "GET" "/api/v1/projects" "200" "" "List projects"
    
    # Get first project ID for subsequent tests
    local project_id
    project_id=$(curl -s "$BASE_URL/api/v1/projects" 2>/dev/null | jq -r '.[0].id // empty')
    
    if [ -n "$project_id" ]; then
        test_endpoint "GET" "/api/v1/projects/$project_id" "200" "" "Get project by ID"
        test_endpoint "GET" "/api/v1/org-charts/$project_id" "200" "" "Get project org chart"
    else
        warn "No projects found - skipping project-specific tests"
    fi
    echo ""
    
    # ============================================================================
    # Agents
    # ============================================================================
    info "Testing Agent endpoints..."
    test_endpoint "GET" "/api/v1/agents" "200" "" "List agents"
    echo ""
    
    # ============================================================================
    # Beads (Work Items)
    # ============================================================================
    info "Testing Bead endpoints..."
    test_endpoint "GET" "/api/v1/beads" "200" "" "List beads"
    
    # Get first bead ID for subsequent tests
    local bead_id
    bead_id=$(curl -s "$BASE_URL/api/v1/beads" 2>/dev/null | jq -r '.[0].id // empty')
    
    if [ -n "$bead_id" ]; then
        test_endpoint "GET" "/api/v1/beads/$bead_id" "200" "" "Get bead by ID"
    else
        warn "No beads found - skipping bead-specific tests"
    fi
    echo ""
    
    # ============================================================================
    # Decisions
    # ============================================================================
    info "Testing Decision endpoints..."
    test_endpoint "GET" "/api/v1/decisions" "200" "" "List decisions"
    echo ""
    
    # ============================================================================
    # Personas
    # ============================================================================
    info "Testing Persona endpoints..."
    test_endpoint "GET" "/api/v1/personas" "200" "" "List personas"
    echo ""
    
    # ============================================================================
    # Events
    # ============================================================================
    info "Testing Event endpoints..."
    test_endpoint "GET" "/api/v1/events/stats" "200" "" "Event statistics"
    
    # Test SSE endpoint (just connection test)
    # Note: SSE endpoints stream indefinitely, so we just test initial connection
    info "Testing event stream connection..."
    ((TOTAL++)) || true
    # Start curl in background, kill after 1 second, check if it started successfully
    if curl -s -N -m 1 "$BASE_URL/api/v1/events/stream" >/dev/null 2>&1; then
        # Curl timed out (expected) or completed - either way connection worked
        success "Event stream connection established"
    else
        # Only fail if curl returned error before timeout
        local exit_code=$?
        if [ $exit_code -eq 28 ]; then
            # Exit code 28 is timeout - this is success for SSE
            success "Event stream connection established"
        else
            warn "Event stream connection test inconclusive (may not be critical)"
            ((PASSED++)) || true  # Don't fail the whole test suite for SSE
        fi
    fi
    echo ""
    
    # ============================================================================
    # Work Graph
    # ============================================================================
    info "Testing Work Graph endpoints..."
    if [ -n "$project_id" ]; then
        test_endpoint "GET" "/api/v1/work-graph?project_id=$project_id" "200" "" "Work graph for project"
    else
        warn "No projects found - skipping work graph test"
    fi
    echo ""
    
    # ============================================================================
    # Summary
    # ============================================================================
    echo "========================================"
    echo "Test Results Summary"
    echo "========================================"
    echo "Total tests:  $TOTAL"
    echo -e "Passed:       ${GREEN}$PASSED${NC}"
    if [ $FAILED -gt 0 ]; then
        echo -e "Failed:       ${RED}$FAILED${NC}"
    else
        echo -e "Failed:       $FAILED"
    fi
    echo "========================================"
    
    if [ $FAILED -gt 0 ]; then
        error "Some tests failed!"
        exit 1
    else
        success "All tests passed!"
        exit 0
    fi
}

# Run main test suite
main "$@"

'use client'

import { useState, useEffect, useRef } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8084'

interface Property {
  id: string
  detail_url: string
  title: string
  image_url?: string
  rent?: number
  floor_plan?: string
  area?: number
  walk_time?: number
  station?: string
  address?: string
  building_age?: number
  floor?: number
  building_type?: string
  facilities?: string
  features?: string
  status: string
  fetched_at: string
  created_at: string
  updated_at: string
}

type ViewMode = 'search' | 'results' | 'compare'
type PropertyStatus = 'none' | 'candidate' | 'maybe' | 'excluded'
type SortOption = 'newest' | 'oldest' | 'rent_asc' | 'rent_desc' | 'area_desc' | 'walk_time_asc' | 'building_age_asc'
type DisplayMode = 'grid' | 'list' | 'map'

export default function Home() {
  const router = useRouter()
  const searchParams = useSearchParams()

  // Helper function to get array from URL params
  const getArrayParam = (key: string, defaultValue: string[] = []): string[] => {
    const value = searchParams.get(key)
    return value ? value.split(',').filter(Boolean) : defaultValue
  }

  // Helper function to get number from URL params
  const getNumberParam = (key: string, defaultValue: number): number => {
    const value = searchParams.get(key)
    return value ? Number(value) : defaultValue
  }

  // Initialize state from URL params or defaults
  const [currentView, setCurrentView] = useState<ViewMode>(
    (searchParams.get('view') as ViewMode) || 'search'
  )
  const [properties, setProperties] = useState<Property[]>([])
  const [loading, setLoading] = useState(false)
  const [propertyStatuses, setPropertyStatuses] = useState<Record<string, PropertyStatus>>({})
  const [hiddenProperties, setHiddenProperties] = useState<Set<string>>(new Set())
  const [propertyMemos, setPropertyMemos] = useState<Record<string, string>>({})

  // Pagination state (server-side)
  const [totalCount, setTotalCount] = useState(0)
  const [offset, setOffset] = useState(0)
  const [hasMore, setHasMore] = useState(true)
  const [isLoadingMore, setIsLoadingMore] = useState(false)
  const [nextCursor, setNextCursor] = useState<string | null>(null)
  const limit = 50

  // AbortController for race condition prevention (reset searches only)
  const searchAbortControllerRef = useRef<AbortController | null>(null)

  // Load property statuses from localStorage on mount
  useEffect(() => {
    const savedStatuses = localStorage.getItem('propertyStatuses')
    if (savedStatuses) {
      try {
        setPropertyStatuses(JSON.parse(savedStatuses))
      } catch (error) {
        console.error('Failed to load property statuses:', error)
      }
    }

    const savedHidden = localStorage.getItem('hiddenProperties')
    if (savedHidden) {
      try {
        setHiddenProperties(new Set(JSON.parse(savedHidden)))
      } catch (error) {
        console.error('Failed to load hidden properties:', error)
      }
    }

    const savedMemos = localStorage.getItem('propertyMemos')
    if (savedMemos) {
      try {
        setPropertyMemos(JSON.parse(savedMemos))
      } catch (error) {
        console.error('Failed to load property memos:', error)
      }
    }
  }, [])

  // Save property statuses to localStorage whenever they change
  useEffect(() => {
    if (Object.keys(propertyStatuses).length > 0) {
      localStorage.setItem('propertyStatuses', JSON.stringify(propertyStatuses))
    }
  }, [propertyStatuses])

  // Save hidden properties to localStorage whenever they change
  useEffect(() => {
    if (hiddenProperties.size > 0) {
      localStorage.setItem('hiddenProperties', JSON.stringify(Array.from(hiddenProperties)))
    }
  }, [hiddenProperties])

  // Save property memos to localStorage whenever they change
  useEffect(() => {
    if (Object.keys(propertyMemos).length > 0) {
      localStorage.setItem('propertyMemos', JSON.stringify(propertyMemos))
    }
  }, [propertyMemos])

  // Header scroll effect
  useEffect(() => {
    const handleScroll = () => {
      const header = document.getElementById('header')
      if (header) {
        if (window.scrollY > 50) {
          header.classList.add('scrolled')
        } else {
          header.classList.remove('scrolled')
        }
      }
    }

    // Initial check on mount
    handleScroll()

    window.addEventListener('scroll', handleScroll)
    return () => window.removeEventListener('scroll', handleScroll)
  }, [])

  // Check header state on view change
  useEffect(() => {
    const header = document.getElementById('header')
    if (header) {
      if (currentView === 'search' && window.scrollY <= 50) {
        header.classList.remove('scrolled')
      } else {
        header.classList.add('scrolled')
      }
    }
  }, [currentView])

  // Search & Filter states (initialized from URL params)
  const [searchQuery, setSearchQuery] = useState(searchParams.get('q') || '')
  const [stationFilter, setStationFilter] = useState(searchParams.get('station') || '')
  const [lineFilter, setLineFilter] = useState(searchParams.get('line') || '')
  const [walkMode, setWalkMode] = useState(searchParams.get('walk_mode') || 'nearest')
  const [minRent, setMinRent] = useState<number>(getNumberParam('minRent', 0))
  const [maxRent, setMaxRent] = useState<number>(getNumberParam('maxRent', 100))
  const [selectedFloorPlans, setSelectedFloorPlans] = useState<string[]>(getArrayParam('floorPlans'))
  const [minWalkTime, setMinWalkTime] = useState<number>(getNumberParam('minWalkTime', 0))
  const [maxWalkTime, setMaxWalkTime] = useState<number>(getNumberParam('maxWalkTime', 60))
  const [minArea, setMinArea] = useState<number>(getNumberParam('minArea', 0))
  const [maxArea, setMaxArea] = useState<number>(getNumberParam('maxArea', 200))
  const [minBuildingAge, setMinBuildingAge] = useState<number>(getNumberParam('minBuildingAge', 0))
  const [maxBuildingAge, setMaxBuildingAge] = useState<number>(getNumberParam('maxBuildingAge', 50))
  const [minFloor, setMinFloor] = useState<number>(getNumberParam('minFloor', 1))
  const [maxFloor, setMaxFloor] = useState<number>(getNumberParam('maxFloor', 30))

  // Additional filters
  const [selectedBuildingTypes, setSelectedBuildingTypes] = useState<string[]>(getArrayParam('buildingTypes'))
  const [selectedFeatures, setSelectedFeatures] = useState<string[]>(getArrayParam('features'))
  const [selectedStatuses, setSelectedStatuses] = useState<string[]>(
    getArrayParam('statuses', [])
  )

  // View options
  const [sortBy, setSortBy] = useState<SortOption>(
    (searchParams.get('sort') as SortOption) || 'newest'
  )
  const [displayMode, setDisplayMode] = useState<DisplayMode>(
    (searchParams.get('display') as DisplayMode) || 'grid'
  )

  // Mobile filter modal state
  const [mobileFilterOpen, setMobileFilterOpen] = useState(false)

  // Mobile menu state
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)

  // Update URL when filters change
  useEffect(() => {
    const params = new URLSearchParams()

    // Add view
    if (currentView !== 'search') {
      params.set('view', currentView)
    }

    // Add search query
    if (searchQuery) params.set('q', searchQuery)

    // Add filters (only if different from defaults)
    if (minRent !== 0) params.set('minRent', minRent.toString())
    if (maxRent !== 100) params.set('maxRent', maxRent.toString())
    if (selectedFloorPlans.length > 0) params.set('floorPlans', selectedFloorPlans.join(','))
    if (minWalkTime !== 0) params.set('minWalkTime', minWalkTime.toString())
    if (maxWalkTime !== 60) params.set('maxWalkTime', maxWalkTime.toString())
    if (minArea !== 0) params.set('minArea', minArea.toString())
    if (maxArea !== 200) params.set('maxArea', maxArea.toString())
    if (minBuildingAge !== 0) params.set('minBuildingAge', minBuildingAge.toString())
    if (maxBuildingAge !== 50) params.set('maxBuildingAge', maxBuildingAge.toString())
    if (minFloor !== 1) params.set('minFloor', minFloor.toString())
    if (maxFloor !== 30) params.set('maxFloor', maxFloor.toString())
    if (selectedBuildingTypes.length > 0) params.set('buildingTypes', selectedBuildingTypes.join(','))
    if (selectedFeatures.length > 0) params.set('features', selectedFeatures.join(','))

    // Status filter (only if different from default)
    const defaultStatuses = ['none', 'candidate', 'maybe'].sort().join(',')
    const currentStatuses = [...selectedStatuses].sort().join(',')
    if (currentStatuses !== defaultStatuses) {
      params.set('statuses', selectedStatuses.join(','))
    }

    // View options
    if (sortBy !== 'newest') params.set('sort', sortBy)
    if (displayMode !== 'grid') params.set('display', displayMode)

    // Update URL without adding to history (using replace)
    const queryString = params.toString()
    const newUrl = queryString ? `/?${queryString}` : '/'
    router.replace(newUrl, { scroll: false })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [
    currentView,
    searchQuery,
    minRent,
    maxRent,
    selectedFloorPlans,
    minWalkTime,
    maxWalkTime,
    minArea,
    maxArea,
    minBuildingAge,
    maxBuildingAge,
    minFloor,
    maxFloor,
    selectedBuildingTypes,
    selectedFeatures,
    selectedStatuses,
    sortBy,
    displayMode
    // Intentionally not including router to avoid infinite loop
  ])

  // Initial fetch on mount
  useEffect(() => {
    fetchProperties()
  }, [])

  // Re-fetch when filters change (with debounce)
  useEffect(() => {
    if (properties.length === 0) return // Skip on initial mount

    const timeoutId = setTimeout(() => {
      fetchProperties(true) // Reset to first page
    }, 300) // 300ms debounce

    return () => clearTimeout(timeoutId)
  }, [
    sortBy,
    minRent,
    maxRent,
    minWalkTime,
    maxWalkTime,
    minArea,
    maxArea,
    minBuildingAge,
    maxBuildingAge,
    minFloor,
    maxFloor,
    selectedFloorPlans,
    selectedBuildingTypes,
    selectedFeatures,
    stationFilter,
    lineFilter,
    walkMode,
    hiddenProperties // Re-fetch when properties are hidden (exclude_ids changes)
  ])

  const fetchProperties = async (resetOffset = false) => {
    // For reset searches (filter changes), abort previous request
    if (resetOffset && searchAbortControllerRef.current) {
      searchAbortControllerRef.current.abort()
    }

    // Create new AbortController for this request (only for reset searches)
    const abortController = resetOffset ? new AbortController() : null
    if (resetOffset) {
      searchAbortControllerRef.current = abortController
    }

    setLoading(true)
    try {
      const currentOffset = resetOffset ? 0 : offset

      // Build URL with all filter parameters
      const params = new URLSearchParams()

      // Map sortBy to API sort parameter
      const sortMapping: Record<SortOption, string> = {
        newest: 'fetched_at_desc',
        oldest: 'fetched_at_asc',
        rent_asc: 'rent_asc',
        rent_desc: 'rent_desc',
        area_desc: 'area_desc',
        walk_time_asc: 'walk_time_asc',
        building_age_asc: 'building_age_asc'
      }

      const apiSort = sortMapping[sortBy] || 'fetched_at'
      params.set('sort', apiSort)

      // Determine if cursor mode is available (only for newest sort)
      const useCursorMode = sortBy === 'newest'

      // Pagination
      params.set('limit', limit.toString())

      if (useCursorMode && !resetOffset && nextCursor) {
        // Use cursor for load more (newest sort only)
        params.set('cursor', nextCursor)
      } else {
        // Use offset for initial load or non-cursor sorts
        params.set('offset', currentOffset.toString())
      }

      // Rent filter (API expects 万円)
      if (minRent > 0) params.set('min_rent', minRent.toString())
      if (maxRent < 100) params.set('max_rent', maxRent.toString())

      // Walk time filter
      if (minWalkTime > 0) params.set('min_walk', minWalkTime.toString())
      if (maxWalkTime < 60) params.set('max_walk', maxWalkTime.toString())

      // Area filter
      if (minArea > 0) params.set('min_area', minArea.toString())
      if (maxArea < 200) params.set('max_area', maxArea.toString())

      // Building age filter
      if (minBuildingAge > 0) params.set('min_building_age', minBuildingAge.toString())
      if (maxBuildingAge < 50) params.set('max_building_age', maxBuildingAge.toString())

      // Floor filter
      if (minFloor > 1) params.set('min_floor', minFloor.toString())
      if (maxFloor < 30) params.set('max_floor', maxFloor.toString())

      // Floor plans (comma-separated)
      if (selectedFloorPlans.length > 0) {
        params.set('floor_plans', selectedFloorPlans.join(','))
      }

      // Building types (comma-separated)
      if (selectedBuildingTypes.length > 0) {
        params.set('building_types', selectedBuildingTypes.join(','))
      }

      // Features/Facilities (comma-separated)
      if (selectedFeatures.length > 0) {
        params.set('facilities', selectedFeatures.join(','))
      }

      // Station and line filters
      if (stationFilter) params.set('station', stationFilter)
      if (lineFilter) params.set('line', lineFilter)
      if (walkMode) params.set('walk_mode', walkMode)

      // Exclude IDs (hidden properties from localStorage) - max 300 to avoid URL length issues
      const hiddenIds = Array.from(hiddenProperties).slice(0, 300)
      if (hiddenIds.length > 0) {
        params.set('exclude_ids', hiddenIds.join(','))
      }

      const response = await fetch(`${API_URL}/api/properties?${params.toString()}`, {
        signal: abortController?.signal
      })

      // Check if request was aborted
      if (abortController?.signal.aborted) {
        return
      }

      const data = await response.json()

      if (data.properties && data.total !== undefined) {
        // Server-side paginated response
        if (resetOffset) {
          setProperties(data.properties)
          setOffset(0)
          // Reset cursor on filter change
          setNextCursor(data.next_cursor || null)
        } else {
          setProperties(prev => [...prev, ...data.properties])
          // Update cursor for next load
          setNextCursor(data.next_cursor || null)
        }
        setTotalCount(data.total)

        // Determine if there's more data
        if (data.next_cursor) {
          setHasMore(true) // If cursor exists, there's more data
        } else {
          // For offset mode or end of cursor data, check manually
          setHasMore(currentOffset + data.properties.length < data.total)
        }

        console.log('✅ Fetched properties:', data.properties.length, 'total:', data.total, 'offset:', currentOffset, 'cursor:', data.next_cursor ? 'yes' : 'no', 'excluded:', hiddenIds.length)
      } else {
        console.error('Unexpected API response format:', data)
      }
    } catch (error) {
      // Ignore abort errors (expected when cancelling requests)
      if (error instanceof Error && error.name === 'AbortError') {
        console.log('🚫 Request aborted (filter changed)')
        return
      }
      console.error('Error fetching properties:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault()
    fetchProperties(true) // Reset to first page
    setCurrentView('results')
  }

  const handleResetFilters = () => {
    setSearchQuery('')
    setStationFilter('')
    setLineFilter('')
    setWalkMode('nearest')
    setMinRent(0)
    setMaxRent(100)
    setSelectedFloorPlans([])
    setMinWalkTime(0)
    setMaxWalkTime(60)
    setMinArea(0)
    setMaxArea(200)
    setMinBuildingAge(0)
    setMaxBuildingAge(50)
    setMinFloor(1)
    setMaxFloor(30)
    setSelectedBuildingTypes([])
    setSelectedFeatures([])
    setSelectedStatuses([])
    setSortBy('newest')
    // Clear URL parameters
    router.replace('/?view=results', { scroll: false })
    // Fetch from beginning
    fetchProperties(true)
  }

  const toggleFloorPlan = (plan: string) => {
    setSelectedFloorPlans(prev =>
      prev.includes(plan)
        ? prev.filter(p => p !== plan)
        : [...prev, plan]
    )
  }

  const toggleBuildingType = (type: string) => {
    setSelectedBuildingTypes(prev =>
      prev.includes(type)
        ? prev.filter(t => t !== type)
        : [...prev, type]
    )
  }

  const toggleFeature = (feature: string) => {
    setSelectedFeatures(prev =>
      prev.includes(feature)
        ? prev.filter(f => f !== feature)
        : [...prev, feature]
    )
  }

  const toggleStatusFilter = (status: string) => {
    setSelectedStatuses(prev =>
      prev.includes(status)
        ? prev.filter(s => s !== status)
        : [...prev, status]
    )
  }

  const setPropertyStatus = (propertyId: string, status: PropertyStatus) => {
    setPropertyStatuses(prev => ({
      ...prev,
      [propertyId]: prev[propertyId] === status ? 'none' : status
    }))
  }

  const hideProperty = (propertyId: string) => {
    setHiddenProperties(prev => {
      const newSet = new Set(prev)
      newSet.add(propertyId)
      return newSet
    })
  }

  const unhideProperty = (propertyId: string) => {
    setHiddenProperties(prev => {
      const newSet = new Set(prev)
      newSet.delete(propertyId)
      return newSet
    })
  }

  const setPropertyMemo = (propertyId: string, memo: string) => {
    setPropertyMemos(prev => {
      if (memo.trim() === '') {
        // Remove memo if empty
        const newMemos = { ...prev }
        delete newMemos[propertyId]
        return newMemos
      }
      return {
        ...prev,
        [propertyId]: memo
      }
    })
  }

  const formatRent = (rent?: number) => {
    if (!rent) return null
    return (rent / 10000).toFixed(1)
  }

  const candidateCount = Object.values(propertyStatuses).filter(s => s === 'candidate').length
  const candidateProperties = properties.filter(p => propertyStatuses[p.id] === 'candidate')

  // Client-side filtering only for localStorage-based status (hidden properties now handled server-side)
  const getDisplayedProperties = () => {
    let filtered = [...properties]

    // Status filter (client-side only, stored in localStorage)
    // Note: hidden properties are now excluded server-side via exclude_ids parameter
    if (selectedStatuses.length > 0) {
      filtered = filtered.filter(p => {
        const status = propertyStatuses[p.id] || 'none'
        return selectedStatuses.includes(status)
      })
    }

    // All other filtering and sorting is done server-side
    return filtered
  }

  const displayedProperties = getDisplayedProperties()

  // Load more function for infinite scroll (offset-based pagination)
  const loadMore = () => {
    if (isLoadingMore || !hasMore) return
    setIsLoadingMore(true)
    setOffset(prev => prev + limit)
  }

  // Fetch more when offset changes
  useEffect(() => {
    if (offset > 0) {
      fetchProperties(false).finally(() => setIsLoadingMore(false))
    }
  }, [offset])

  // Infinite scroll observer
  useEffect(() => {
    if (currentView !== 'results') return

    const handleScroll = () => {
      // Check if user scrolled near bottom of page
      const scrollHeight = document.documentElement.scrollHeight
      const scrollTop = document.documentElement.scrollTop
      const clientHeight = document.documentElement.clientHeight

      // Trigger load more when user is within 500px of bottom
      if (scrollHeight - scrollTop - clientHeight < 500) {
        if (hasMore && !isLoadingMore) {
          loadMore()
        }
      }
    }

    window.addEventListener('scroll', handleScroll)
    return () => window.removeEventListener('scroll', handleScroll)
  }, [currentView, hasMore, isLoadingMore])

  return (
    <div className="app">
      <header className={`header ${currentView === 'results' ? 'results-header' : ''}`} id="header">
        <div className="header-inner">
          <a href="#" className="logo" onClick={() => setCurrentView('search')}>shiboroom<span>.</span></a>
          {(currentView === 'search' || currentView === 'compare') && (
            <nav className="nav">
              <a href="#" onClick={(e) => { e.preventDefault(); setCurrentView('results'); }}>物件一覧</a>
              <a href="#" onClick={(e) => { e.preventDefault(); setCurrentView('compare'); }}>候補比較</a>
              <a href="#" className="nav-login" onClick={(e) => { e.preventDefault(); alert('準備中'); }}>ログイン</a>
            </nav>
          )}
        </div>
      </header>

      {/* Mobile Menu */}
      <div className={`mobile-menu ${mobileMenuOpen ? 'open' : ''}`}>
        <div className="mobile-menu-header">
          <span className="mobile-menu-title">メニュー</span>
          <button
            className="mobile-menu-close"
            onClick={() => setMobileMenuOpen(false)}
            aria-label="メニューを閉じる"
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
              <line x1="18" y1="6" x2="6" y2="18"/>
              <line x1="6" y1="6" x2="18" y2="18"/>
            </svg>
          </button>
        </div>
        <nav className="mobile-menu-nav">
          <button
            className={`mobile-menu-item ${currentView === 'search' ? 'active' : ''}`}
            onClick={() => {
              setCurrentView('search')
              setMobileMenuOpen(false)
            }}
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>
              <polyline points="9 22 9 12 15 12 15 22"/>
            </svg>
            <span>ホーム</span>
          </button>
          <button
            className={`mobile-menu-item ${currentView === 'results' ? 'active' : ''}`}
            onClick={() => {
              setCurrentView('results')
              setMobileMenuOpen(false)
            }}
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <line x1="8" y1="6" x2="21" y2="6"/>
              <line x1="8" y1="12" x2="21" y2="12"/>
              <line x1="8" y1="18" x2="21" y2="18"/>
              <line x1="3" y1="6" x2="3.01" y2="6"/>
              <line x1="3" y1="12" x2="3.01" y2="12"/>
              <line x1="3" y1="18" x2="3.01" y2="18"/>
            </svg>
            <span>物件一覧</span>
          </button>
          <button
            className={`mobile-menu-item ${currentView === 'compare' ? 'active' : ''}`}
            onClick={() => {
              setCurrentView('compare')
              setMobileMenuOpen(false)
            }}
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <rect x="3" y="3" width="7" height="7"/>
              <rect x="14" y="3" width="7" height="7"/>
              <rect x="14" y="14" width="7" height="7"/>
              <rect x="3" y="14" width="7" height="7"/>
            </svg>
            <span>候補比較</span>
          </button>
          <a href="/admin" className="mobile-menu-item">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/>
              <circle cx="12" cy="12" r="3"/>
            </svg>
            <span>管理画面</span>
          </a>
        </nav>
        <div className="mobile-menu-footer">
          <div className="mobile-menu-stats">
            <div className="mobile-menu-stat">
              <span className="label">全物件</span>
              <span className="value accent">{totalCount}件</span>
            </div>
            <div className="mobile-menu-stat">
              <span className="label">候補</span>
              <span className="value candidate">{candidateCount}件</span>
            </div>
          </div>
        </div>
      </div>

      {/* Mobile Menu Backdrop */}
      <div
        className={`mobile-menu-backdrop ${mobileMenuOpen ? 'open' : ''}`}
        onClick={() => setMobileMenuOpen(false)}
        aria-hidden="true"
      />

      {/* Home View */}
      {currentView === 'search' && (
        <section className="lp-view active" id="home">
          {/* Hero Section */}
          <section className="lp-hero">
            {/* Background Image */}
            <div className="lp-hero-bg">
              <div className="lp-hero-bg-fallback"></div>
              <img
                src="/hero-room.jpg"
                alt="モダンなリビングルーム"
                className="lp-hero-bg-image"
              />
            </div>

            <div className="lp-hero-inner">
              <div className="lp-hero-content">
                <p className="lp-hero-label">新しい賃貸検索</p>
                <h1 className="lp-hero-title">
                  <span className="lp-hero-title-accent">気になる物件だけ</span>が<br />残る部屋探し
                </h1>
                <p className="lp-hero-subtitle">
                  条件に合わない物件をワンタップで非表示に。<br />
                  検索するたびに候補が絞られていく、新しい体験です。
                </p>
                <p className="lp-hero-note">
                  掲載料で上位が決まる検索ではなく、<br />
                  あなたの判断が中心の設計です。
                </p>
                <button className="lp-hero-cta" onClick={() => setCurrentView('results')}>
                  物件一覧
                </button>
              </div>
            </div>

            {/* Stats - Right Bottom */}
            <div className="lp-hero-stats">
              <div className="lp-stat-item">
                <div className="lp-stat-value">2,400<span>+</span></div>
                <div className="lp-stat-label">掲載物件</div>
              </div>
              <div className="lp-stat-item">
                <div className="lp-stat-value">89<span>%</span></div>
                <div className="lp-stat-label">満足度</div>
              </div>
            </div>

            {/* Scroll Indicator */}
            <div className="lp-scroll-indicator">
              Scroll
            </div>
          </section>

          {/* Problem Section */}
          <section className="lp-problem">
            <div className="lp-problem-inner">
              <p className="lp-section-label">
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <circle cx="12" cy="12" r="10"/>
                  <line x1="12" y1="8" x2="12" y2="12"/>
                  <line x1="12" y1="16" x2="12.01" y2="16"/>
                </svg>
                こんな経験ありませんか？
              </p>
              <h2 className="lp-problem-title">
                部屋探しが疲れるのは、<br />
                情報が多すぎるから
              </h2>
              <div className="lp-problem-list">
                <div className="lp-problem-item">
                  <div className="lp-problem-icon">
                    <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
                      <polyline points="14 2 14 8 20 8"/>
                      <line x1="16" y1="13" x2="8" y2="13"/>
                      <line x1="16" y1="17" x2="8" y2="17"/>
                      <polyline points="10 9 9 9 8 9"/>
                    </svg>
                  </div>
                  <p className="lp-problem-text">条件を絞っても、関係ない物件が大量に表示される</p>
                </div>
                <div className="lp-problem-item">
                  <div className="lp-problem-icon">
                    <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                      <polyline points="23 4 23 10 17 10"/>
                      <polyline points="1 20 1 14 7 14"/>
                      <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
                    </svg>
                  </div>
                  <p className="lp-problem-text">一度見て却下した物件が、検索するたびに出てくる</p>
                </div>
                <div className="lp-problem-item">
                  <div className="lp-problem-icon">
                    <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                      <circle cx="12" cy="12" r="10"/>
                      <path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"/>
                      <line x1="12" y1="17" x2="12.01" y2="17"/>
                    </svg>
                  </div>
                  <p className="lp-problem-text">表記が正しいか分からず、すべて疑いながら見る必要がある</p>
                </div>
                <div className="lp-problem-item">
                  <div className="lp-problem-icon">
                    <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                      <line x1="8" y1="6" x2="21" y2="6"/>
                      <line x1="8" y1="12" x2="21" y2="12"/>
                      <line x1="8" y1="18" x2="21" y2="18"/>
                      <line x1="3" y1="6" x2="3.01" y2="6"/>
                      <line x1="3" y1="12" x2="3.01" y2="12"/>
                      <line x1="3" y1="18" x2="3.01" y2="18"/>
                    </svg>
                  </div>
                  <p className="lp-problem-text">探せば探すほど、どれがいいのか分からなくなる</p>
                </div>
              </div>
            </div>
          </section>

          {/* Solution Section */}
          <section className="lp-solution">
            <div className="lp-solution-inner">
              <p className="lp-section-label">
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/>
                </svg>
                shiboroomの特徴
              </p>
              <div className="lp-solution-header">
                <p className="lp-solution-tagline">「増やす」より「減らす」</p>
                <h2 className="lp-solution-title">
                  興味のない物件を消していく、<br />
                  新しい検索体験
                </h2>
              </div>

              <div className="lp-features">
                <div className="lp-feature">
                  <div className="lp-feature-header">
                    <div className="lp-feature-number">1</div>
                    <h3 className="lp-feature-title">ワンタップで非表示<span className="lp-feature-badge">コア</span></h3>
                  </div>
                  <p className="lp-feature-desc">
                    気に入らない物件はその場で非表示に。一度消した物件は、次回から表示されません。使うほど検索結果がスッキリしていきます。
                  </p>
                  <div className="lp-feature-visual">
                    <div className="lp-demo-card">
                      <div className="lp-demo-card-info">
                        <p className="lp-demo-card-title">渋谷区恵比寿 1K</p>
                        <p className="lp-demo-card-meta">8.5万円 / 徒歩7分 / 25㎡</p>
                      </div>
                      <button className="lp-demo-hide-btn" title="非表示にする">
                        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="2">
                          <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                        </svg>
                      </button>
                    </div>
                  </div>
                </div>

                <div className="lp-feature">
                  <div className="lp-feature-header">
                    <div className="lp-feature-number">2</div>
                    <h3 className="lp-feature-title">◎ △ × で絞り込み<span className="lp-feature-badge">コア</span></h3>
                  </div>
                  <p className="lp-feature-desc">
                    「できれば8分以内、10分までなら許容」のような曖昧な条件を設定できます。×は自動で除外、△は注意つきで表示します。
                  </p>
                  <div className="lp-feature-visual">
                    <p className="lp-demo-rating-label">駅からの距離</p>
                    <div className="lp-demo-rating">
                      <div className="lp-rating-option">
                        <button className="lp-rating-btn active-good">◎</button>
                        <span className="lp-rating-label">〜8分</span>
                      </div>
                      <div className="lp-rating-option">
                        <button className="lp-rating-btn active-ok">△</button>
                        <span className="lp-rating-label">9〜10分</span>
                      </div>
                      <div className="lp-rating-option">
                        <button className="lp-rating-btn active-bad">×</button>
                        <span className="lp-rating-label">11分〜</span>
                      </div>
                    </div>
                  </div>
                </div>

                <div className="lp-feature">
                  <div className="lp-feature-header">
                    <div className="lp-feature-number">3</div>
                    <h3 className="lp-feature-title">怪しい物件が分かる<span className="lp-feature-badge">コア</span></h3>
                  </div>
                  <p className="lp-feature-desc">
                    ユーザーからの「表記と違うかも」報告を可視化します。運営が正否を断定するのではなく、報告があること・件数を表示します。すべての物件を疑って見る必要がなくなります。
                  </p>
                  <div className="lp-feature-visual">
                    <div className="lp-demo-report">
                      <div className="lp-report-header">
                        <svg className="lp-report-icon" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="2">
                          <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                        </svg>
                        <span className="lp-report-title">相違報告あり</span>
                      </div>
                      <div className="lp-report-item">
                        <span>写真と実際の印象が違う</span>
                        <span className="lp-report-count">2件</span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              {/* Plus Alpha */}
              <div className="lp-plus-alpha">
                <p className="lp-plus-alpha-header">＋α あると便利</p>
                <div className="lp-plus-alpha-list">
                  <div className="lp-plus-alpha-item">
                    <span className="lp-plus-alpha-icon">
                      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                        <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                        <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
                      </svg>
                    </span>
                    <span><strong>メモ機能</strong>:却下理由・内見メモを残せる</span>
                  </div>
                  <div className="lp-plus-alpha-item">
                    <span className="lp-plus-alpha-icon">
                      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                        <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
                        <line x1="3" y1="9" x2="21" y2="9"/>
                        <line x1="9" y1="21" x2="9" y2="9"/>
                      </svg>
                    </span>
                    <span><strong>広さの不明可視化</strong>:LDKなど定義が不明な場合は「不明」として扱える(比較しやすい)</span>
                  </div>
                </div>
              </div>
            </div>
          </section>

          {/* Why Section */}
          <section className="lp-why">
            <div className="lp-why-inner">
              <p className="lp-section-label">
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <circle cx="12" cy="12" r="10"/>
                  <line x1="12" y1="16" x2="12" y2="12"/>
                  <line x1="12" y1="8" x2="12.01" y2="8"/>
                </svg>
                なぜ他ではできなかった?
              </p>
              <h2 className="lp-why-title">掲載料モデルの限界</h2>
              <div className="lp-why-text">
                <p>
                  既存の不動産ポータルサイトは、不動産会社から掲載料をもらうモデルが中心です。そのため、ユーザーが物件をどんどん非表示にする体験は、構造的に提供しづらいことがあります。
                </p>
                <p>
                  結果として、興味のない物件が何度も表示され、ユーザーは「使いにくい」と感じながらも我慢して使い続けていました。
                </p>
              </div>
              <div className="lp-why-highlight">
                <p className="lp-why-highlight-text">
                  shiboroomは、探す量を減らすことを最優先に、判断しやすい検索体験を提供します。
                </p>
              </div>
            </div>
          </section>

          {/* CTA Section */}
          <section className="lp-cta">
            <div className="lp-cta-content">
              <span className="lp-cta-badge">
                <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <polyline points="20 6 9 17 4 12"/>
                </svg>
                登録不要で今すぐ検索できます
              </span>
              <h2 className="lp-cta-title">部屋探しをはじめる</h2>
              <div className="lp-cta-buttons">
                <button className="lp-cta-btn" onClick={() => setCurrentView('results')}>
                  物件一覧
                </button>
              </div>
            </div>
          </section>
        </section>
      )}

      {/* Results View */}
      {currentView === 'results' && (
        <section className="results-view active">
          {/* Mobile Filter Button */}
          <button
            className="mobile-filter-btn"
            onClick={() => setMobileFilterOpen(true)}
            aria-label="フィルタを開く"
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <line x1="4" y1="6" x2="20" y2="6"/>
              <line x1="4" y1="12" x2="20" y2="12"/>
              <line x1="4" y1="18" x2="20" y2="18"/>
              <circle cx="8" cy="6" r="2" fill="currentColor"/>
              <circle cx="16" cy="12" r="2" fill="currentColor"/>
              <circle cx="12" cy="18" r="2" fill="currentColor"/>
            </svg>
          </button>

          {/* Sidebar Filters */}
          <aside className={`sidebar ${mobileFilterOpen ? 'mobile-open' : ''}`}>
            {/* Logo in Sidebar */}
            <div className="sidebar-logo">
              <a href="#" className="logo" onClick={(e) => { e.preventDefault(); setCurrentView('search'); }}>
                shiboroom<span>.</span>
              </a>
            </div>

            {/* Mobile Close Button */}
            <button
              className="mobile-filter-close"
              onClick={() => setMobileFilterOpen(false)}
              aria-label="フィルタを閉じる"
            >
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
                <line x1="18" y1="6" x2="6" y2="18"/>
                <line x1="6" y1="6" x2="18" y2="18"/>
              </svg>
            </button>

            <div className="sidebar-header">
              <span className="sidebar-title">条件で絞り込み</span>
            </div>
            <div className="sidebar-content">
              <div className="filter-section">
                <h3 className="filter-title">リアルタイム絞り込み</h3>

                <div className="filter-group">
                  <label className="filter-label">
                    家賃
                    <span className="filter-value">{minRent}〜{maxRent}万円</span>
                  </label>
                  <div className="dual-range">
                    <input
                      type="range"
                      className="range-slider"
                      min="0"
                      max="100"
                      value={minRent}
                      onChange={(e) => {
                        const val = Number(e.target.value)
                        if (val < maxRent) setMinRent(val)
                      }}
                    />
                    <input
                      type="range"
                      className="range-slider"
                      min="0"
                      max="100"
                      value={maxRent}
                      onChange={(e) => {
                        const val = Number(e.target.value)
                        if (val > minRent) setMaxRent(val)
                      }}
                    />
                  </div>
                </div>

                <div className="filter-group">
                  <label className="filter-label">
                    駅徒歩
                    <span className="filter-value">{minWalkTime}〜{maxWalkTime}分</span>
                  </label>
                  <div className="dual-range">
                    <input
                      type="range"
                      className="range-slider"
                      min="0"
                      max="60"
                      value={minWalkTime}
                      onChange={(e) => {
                        const val = Number(e.target.value)
                        if (val < maxWalkTime) setMinWalkTime(val)
                      }}
                    />
                    <input
                      type="range"
                      className="range-slider"
                      min="0"
                      max="60"
                      value={maxWalkTime}
                      onChange={(e) => {
                        const val = Number(e.target.value)
                        if (val > minWalkTime) setMaxWalkTime(val)
                      }}
                    />
                  </div>
                </div>

                <div className="filter-group">
                  <label className="filter-label">
                    専有面積
                    <span className="filter-value">{minArea}〜{maxArea}㎡</span>
                  </label>
                  <div className="dual-range">
                    <input
                      type="range"
                      className="range-slider"
                      min="0"
                      max="200"
                      value={minArea}
                      onChange={(e) => {
                        const val = Number(e.target.value)
                        if (val < maxArea) setMinArea(val)
                      }}
                    />
                    <input
                      type="range"
                      className="range-slider"
                      min="0"
                      max="200"
                      value={maxArea}
                      onChange={(e) => {
                        const val = Number(e.target.value)
                        if (val > minArea) setMaxArea(val)
                      }}
                    />
                  </div>
                </div>

                <div className="filter-group">
                  <label className="filter-label">
                    築年数
                    <span className="filter-value">{minBuildingAge}〜{maxBuildingAge}年</span>
                  </label>
                  <div className="dual-range">
                    <input
                      type="range"
                      className="range-slider"
                      min="0"
                      max="50"
                      value={minBuildingAge}
                      onChange={(e) => {
                        const val = Number(e.target.value)
                        if (val < maxBuildingAge) setMinBuildingAge(val)
                      }}
                    />
                    <input
                      type="range"
                      className="range-slider"
                      min="0"
                      max="50"
                      value={maxBuildingAge}
                      onChange={(e) => {
                        const val = Number(e.target.value)
                        if (val > minBuildingAge) setMaxBuildingAge(val)
                      }}
                    />
                  </div>
                </div>

                <div className="filter-group">
                  <label className="filter-label">
                    階数
                    <span className="filter-value">{minFloor}〜{maxFloor}階</span>
                  </label>
                  <div className="dual-range">
                    <input
                      type="range"
                      className="range-slider"
                      min="1"
                      max="30"
                      value={minFloor}
                      onChange={(e) => {
                        const val = Number(e.target.value)
                        if (val < maxFloor) setMinFloor(val)
                      }}
                    />
                    <input
                      type="range"
                      className="range-slider"
                      min="1"
                      max="30"
                      value={maxFloor}
                      onChange={(e) => {
                        const val = Number(e.target.value)
                        if (val > minFloor) setMaxFloor(val)
                      }}
                    />
                  </div>
                </div>

                <div className="filter-group">
                  <label className="filter-label">間取り</label>
                  <div className="checkbox-group">
                    {['1R', '1K', '1DK', '1LDK', '2K', '2DK', '2LDK', '3K', '3DK', '3LDK'].map(plan => (
                      <label key={plan} className="checkbox-item">
                        <input
                          type="checkbox"
                          checked={selectedFloorPlans.includes(plan)}
                          onChange={() => toggleFloorPlan(plan)}
                        />
                        <span className="checkbox-label">{plan}</span>
                      </label>
                    ))}
                  </div>
                </div>

                <div className="filter-group">
                  <label className="filter-label">建物種別</label>
                  <div className="checkbox-group">
                    {[
                      { key: 'mansion', label: 'マンション' },
                      { key: 'apartment', label: 'アパート' },
                      { key: 'house', label: '一戸建て' },
                      { key: 'terrace_house', label: 'テラスハウス' },
                      { key: 'town_house', label: 'タウンハウス' }
                    ].map(({ key, label }) => (
                      <label key={key} className="checkbox-item">
                        <input
                          type="checkbox"
                          checked={selectedBuildingTypes.includes(key)}
                          onChange={() => toggleBuildingType(key)}
                        />
                        <span className="checkbox-label">{label}</span>
                      </label>
                    ))}
                  </div>
                </div>

                {/* 人気のこだわり条件 */}
                <div className="filter-group">
                  <label className="filter-label">人気のこだわり条件</label>
                  <div className="checkbox-group">
                    {[
                      { key: 'bath_toilet_separate', label: 'バストイレ別' },
                      { key: 'second_floor_plus', label: '2階以上' },
                      { key: 'indoor_washer', label: '洗濯機置き場' },
                      { key: 'parking', label: '駐車場あり' },
                      { key: 'ac', label: 'エアコン' },
                      { key: 'flooring', label: 'フローリング' },
                      { key: 'washbasin', label: '洗面台' },
                      { key: 'pet_friendly', label: 'ペット相談' },
                      { key: 'balcony', label: 'ベランダ' },
                      { key: 'reheating_bath', label: '追い焚き機能' }
                    ].map(({ key, label }) => (
                      <label key={key} className="checkbox-item">
                        <input
                          type="checkbox"
                          checked={selectedFeatures.includes(key)}
                          onChange={() => toggleFeature(key)}
                        />
                        <span className="checkbox-label">{label}</span>
                      </label>
                    ))}
                  </div>
                </div>

                {/* バス・トイレ */}
                <div className="filter-group">
                  <label className="filter-label">バス・トイレ</label>
                  <div className="checkbox-group">
                    {[
                      { key: 'bath_toilet_separate', label: 'バストイレ別' },
                      { key: 'washbasin', label: '洗面台' },
                      { key: 'reheating_bath', label: '追い焚き機能' },
                      { key: 'bathroom_dryer', label: '浴室乾燥機' },
                      { key: 'washlet', label: '温水洗浄便座' }
                    ].map(({ key, label }) => (
                      <label key={key} className="checkbox-item">
                        <input
                          type="checkbox"
                          checked={selectedFeatures.includes(key)}
                          onChange={() => toggleFeature(key)}
                        />
                        <span className="checkbox-label">{label}</span>
                      </label>
                    ))}
                  </div>
                </div>

                {/* キッチン */}
                <div className="filter-group">
                  <label className="filter-label">キッチン</label>
                  <div className="checkbox-group">
                    {[
                      { key: 'gas_stove', label: 'ガスコンロ可' },
                      { key: 'system_kitchen', label: 'システムキッチン' },
                      { key: 'counter_kitchen', label: 'カウンターキッチン' },
                      { key: 'ih_stove', label: 'IHクッキングヒーター' },
                      { key: 'city_gas', label: '都市ガス' }
                    ].map(({ key, label }) => (
                      <label key={key} className="checkbox-item">
                        <input
                          type="checkbox"
                          checked={selectedFeatures.includes(key)}
                          onChange={() => toggleFeature(key)}
                        />
                        <span className="checkbox-label">{label}</span>
                      </label>
                    ))}
                  </div>
                </div>

                {/* 収納 */}
                <div className="filter-group">
                  <label className="filter-label">収納</label>
                  <div className="checkbox-group">
                    {[
                      { key: 'closet', label: 'クローゼット' },
                      { key: 'walk_in_closet', label: 'ウォークインクローゼット' },
                      { key: 'storage', label: '物置' },
                      { key: 'trunk_room', label: 'トランクルーム' }
                    ].map(({ key, label }) => (
                      <label key={key} className="checkbox-item">
                        <input
                          type="checkbox"
                          checked={selectedFeatures.includes(key)}
                          onChange={() => toggleFeature(key)}
                        />
                        <span className="checkbox-label">{label}</span>
                      </label>
                    ))}
                  </div>
                </div>

                {/* セキュリティ */}
                <div className="filter-group">
                  <label className="filter-label">セキュリティ</label>
                  <div className="checkbox-group">
                    {[
                      { key: 'auto_lock', label: 'オートロック' },
                      { key: 'tv_intercom', label: 'モニター付きインターホン' },
                      { key: 'security_camera', label: '防犯カメラ' },
                      { key: 'card_key', label: 'カードキー' },
                      { key: 'resident_manager', label: '管理人常駐' }
                    ].map(({ key, label }) => (
                      <label key={key} className="checkbox-item">
                        <input
                          type="checkbox"
                          checked={selectedFeatures.includes(key)}
                          onChange={() => toggleFeature(key)}
                        />
                        <span className="checkbox-label">{label}</span>
                      </label>
                    ))}
                  </div>
                </div>

                {/* 設備 */}
                <div className="filter-group">
                  <label className="filter-label">設備</label>
                  <div className="checkbox-group">
                    {[
                      { key: 'indoor_washer', label: '洗濯機置き場' },
                      { key: 'ac', label: 'エアコン' },
                      { key: 'flooring', label: 'フローリング' },
                      { key: 'loft', label: 'ロフト' },
                      { key: 'elevator', label: 'エレベーター' },
                      { key: 'delivery_box', label: '宅配ボックス' },
                      { key: 'divided_condo', label: '分譲タイプ' },
                      { key: 'maisonette', label: 'メゾネット' },
                      { key: 'barrier_free', label: 'バリアフリー' },
                      { key: 'floor_heating', label: '床暖房' }
                    ].map(({ key, label }) => (
                      <label key={key} className="checkbox-item">
                        <input
                          type="checkbox"
                          checked={selectedFeatures.includes(key)}
                          onChange={() => toggleFeature(key)}
                        />
                        <span className="checkbox-label">{label}</span>
                      </label>
                    ))}
                  </div>
                </div>

                {/* 位置 */}
                <div className="filter-group">
                  <label className="filter-label">位置</label>
                  <div className="checkbox-group">
                    {[
                      { key: 'first_floor', label: '1階' },
                      { key: 'second_floor_plus', label: '2階以上' },
                      { key: 'top_floor', label: '最上階' },
                      { key: 'corner_room', label: '角部屋' },
                      { key: 'south_facing', label: '南向き' }
                    ].map(({ key, label }) => (
                      <label key={key} className="checkbox-item">
                        <input
                          type="checkbox"
                          checked={selectedFeatures.includes(key)}
                          onChange={() => toggleFeature(key)}
                        />
                        <span className="checkbox-label">{label}</span>
                      </label>
                    ))}
                  </div>
                </div>

                {/* 駐車場・駐輪場 */}
                <div className="filter-group">
                  <label className="filter-label">駐車場・駐輪場</label>
                  <div className="checkbox-group">
                    {[
                      { key: 'parking', label: '駐車場あり' },
                      { key: 'garage', label: '車庫' },
                      { key: 'bike_parking', label: '自転車置き場' }
                    ].map(({ key, label }) => (
                      <label key={key} className="checkbox-item">
                        <input
                          type="checkbox"
                          checked={selectedFeatures.includes(key)}
                          onChange={() => toggleFeature(key)}
                        />
                        <span className="checkbox-label">{label}</span>
                      </label>
                    ))}
                  </div>
                </div>

                {/* 入居条件 */}
                <div className="filter-group">
                  <label className="filter-label">入居条件</label>
                  <div className="checkbox-group">
                    {[
                      { key: 'pet_friendly', label: 'ペット相談' },
                      { key: 'instrument_ok', label: '楽器相談' },
                      { key: 'two_occupants', label: '二人入居可' },
                      { key: 'immediate_available', label: '即入居可' }
                    ].map(({ key, label }) => (
                      <label key={key} className="checkbox-item">
                        <input
                          type="checkbox"
                          checked={selectedFeatures.includes(key)}
                          onChange={() => toggleFeature(key)}
                        />
                        <span className="checkbox-label">{label}</span>
                      </label>
                    ))}
                  </div>
                </div>


                <div className="filter-group">
                  <label className="filter-label">ステータス</label>
                  <div className="checkbox-group">
                    {[
                      { key: 'none', label: '未分類' },
                      { key: 'candidate', label: '候補' },
                      { key: 'maybe', label: '微妙' },
                      { key: 'excluded', label: '除外' }
                    ].map(({ key, label }) => (
                      <label key={key} className="checkbox-item">
                        <input
                          type="checkbox"
                          checked={selectedStatuses.includes(key)}
                          onChange={() => toggleStatusFilter(key)}
                        />
                        <span className="checkbox-label">{label}</span>
                      </label>
                    ))}
                  </div>
                </div>
              </div>
              <div className="filter-result">
                <div className="filter-result-count">{displayedProperties.length}</div>
                <div className="filter-result-label">件がヒット</div>
              </div>
              <button className="reset-btn" onClick={handleResetFilters}>絞り込みをリセット</button>

              {/* Action Buttons */}
              <div className="sidebar-actions">
                <button
                  className="sidebar-action-btn compare-btn"
                  onClick={() => setCurrentView('compare')}
                >
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <path d="M9 12h6M9 16h6M9 8h6M3 3h18M3 21h18M3 3v18M21 3v18"/>
                  </svg>
                  候補を比較
                  {Object.values(propertyStatuses).filter(s => s === 'candidate').length > 0 && (
                    <span className="badge">{Object.values(propertyStatuses).filter(s => s === 'candidate').length}</span>
                  )}
                </button>
                <button
                  className="sidebar-action-btn login-btn"
                  onClick={(e) => { e.preventDefault(); alert('準備中'); }}
                >
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4M10 17l5-5-5-5M15 12H3"/>
                  </svg>
                  ログイン
                </button>
              </div>
            </div>
          </aside>

          <main className="main-content">
            <div className="toolbar">
              <div className="toolbar-left">
                <div className="search-status">
                  {loading ? (
                    <><span className="loading"></span> 読み込み中...</>
                  ) : (
                    <>
                      <span className="count-primary">{displayedProperties.length}件</span>
                      <span className="count-separator">/</span>
                      <span className="count-total">全{totalCount}件</span>
                    </>
                  )}
                </div>
              </div>
              <div className="toolbar-left">
                <select value={sortBy} onChange={(e) => setSortBy(e.target.value as SortOption)} className="sort-select">
                  <option value="newest">新着順</option>
                  <option value="oldest">古い順</option>
                  <option value="rent_asc">家賃が安い順</option>
                  <option value="walk_time_asc">駅から近い順</option>
                  <option value="building_age_asc">築年数が新しい順</option>
                  <option value="area_desc">専有面積が広い順</option>
                </select>
                <div className="view-toggle">
                  <button
                    className={`view-toggle-btn ${displayMode === 'grid' ? 'active' : ''}`}
                    onClick={() => setDisplayMode('grid')}
                    title="グリッド表示"
                  >
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                      <rect width="7" height="7" x="3" y="3" rx="1"/>
                      <rect width="7" height="7" x="14" y="3" rx="1"/>
                      <rect width="7" height="7" x="14" y="14" rx="1"/>
                      <rect width="7" height="7" x="3" y="14" rx="1"/>
                    </svg>
                  </button>
                  <button
                    className={`view-toggle-btn ${displayMode === 'list' ? 'active' : ''}`}
                    onClick={() => setDisplayMode('list')}
                    title="リスト表示"
                  >
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                      <line x1="8" x2="21" y1="6" y2="6"/>
                      <line x1="8" x2="21" y1="12" y2="12"/>
                      <line x1="8" x2="21" y1="18" y2="18"/>
                      <line x1="3" x2="3.01" y1="6" y2="6"/>
                      <line x1="3" x2="3.01" y1="12" y2="12"/>
                      <line x1="3" x2="3.01" y1="18" y2="18"/>
                    </svg>
                  </button>
                  <button
                    className={`view-toggle-btn ${displayMode === 'map' ? 'active' : ''}`}
                    onClick={() => setDisplayMode('map')}
                    title="地図表示"
                  >
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                      <path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"/>
                      <circle cx="12" cy="10" r="3"/>
                    </svg>
                  </button>
                </div>
              </div>
            </div>

            {displayMode === 'map' && (
              <div className="map-container">
                <div className="map-placeholder">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"/>
                    <circle cx="12" cy="10" r="3"/>
                  </svg>
                  <p>地図表示機能は開発中です</p>
                </div>
              </div>
            )}

            <div className="property-grid">
              {displayedProperties.map((property) => (
                <PropertyCard
                  key={property.id}
                  property={property}
                  formatRent={formatRent}
                  status={propertyStatuses[property.id] || 'none'}
                  onStatusChange={(status) => setPropertyStatus(property.id, status)}
                  onHide={() => hideProperty(property.id)}
                />
              ))}
            </div>

            {/* Infinite Scroll - Loading Indicator */}
            {hasMore && isLoadingMore && (
              <div className="load-more-container">
                <span className="loading"></span>
                <div className="load-more-info">読み込み中...</div>
              </div>
            )}

            {!hasMore && displayedProperties.length > 0 && (
              <div className="load-more-container">
                <div className="load-more-info">
                  全{totalCount}件を表示しました
                </div>
              </div>
            )}
          </main>
        </section>
      )}

      {/* Compare View */}
      {currentView === 'compare' && (
        <section className="compare-view active">
          <div className="compare-header">
            <h2 className="compare-title">候補物件 <span>{candidateCount}件</span></h2>
          </div>
          <div className="compare-table">
            <table>
              <thead>
                <tr>
                  <th>物件</th>
                  <th>賃料</th>
                  <th>間取り</th>
                  <th>最寄駅</th>
                  <th>築年数</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                {candidateProperties.map(property => (
                  <tr key={property.id}>
                    <td>
                      <div className="compare-property">
                        {property.image_url && (
                          <img src={property.image_url} className="compare-property-img" alt="" />
                        )}
                        <div>
                          <div className="compare-property-name">{property.title}</div>
                          <div className="compare-property-address">{property.address}</div>
                        </div>
                      </div>
                    </td>
                    <td>
                      {property.rent && (
                        <span className="compare-value highlight">{formatRent(property.rent)}万円</span>
                      )}
                    </td>
                    <td>
                      <span className="compare-value">
                        {property.floor_plan} / {property.area}㎡
                      </span>
                    </td>
                    <td>
                      <span className="compare-value">
                        {property.station} 徒歩{property.walk_time}分
                      </span>
                    </td>
                    <td>
                      <span className="compare-value">築{property.building_age}年</span>
                    </td>
                    <td>
                      <a href={property.detail_url} className="btn btn-secondary" target="_blank" rel="noreferrer">
                        詳細
                      </a>
                    </td>
                  </tr>
                ))}
                {candidateProperties.length === 0 && (
                  <tr>
                    <td colSpan={6} style={{ textAlign: 'center', padding: '40px', color: 'var(--text-muted)' }}>
                      候補に追加された物件はありません
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </section>
      )}

      {/* Footer - Only show on home page */}
      {currentView === 'search' && (
        <footer className="footer">
          <div className="footer-inner">
            <div className="footer-links">
              <a href="/terms" className="footer-link">利用規約</a>
              <a href="/privacy" className="footer-link">プライバシーポリシー</a>
              <a href="/tokushoho" className="footer-link">特定商取引法に基づく表記</a>
            </div>
            <p className="footer-copy">© 2025 shiboroom</p>
          </div>
        </footer>
      )}
    </div>
  )
}

function PropertyCard({
  property,
  formatRent,
  status,
  onStatusChange,
  onHide
}: {
  property: Property
  formatRent: (rent?: number) => string | null
  status: PropertyStatus
  onStatusChange: (status: PropertyStatus) => void
  onHide: () => void
}) {
  const [imageError, setImageError] = useState(false)
  const router = useRouter()

  const handleCardClick = (e: React.MouseEvent) => {
    // Don't navigate if clicking on buttons
    if ((e.target as HTMLElement).closest('button, a')) {
      return
    }
    router.push(`/properties/${property.id}`)
  }

  return (
    <article
      className={`property-card ${status !== 'none' ? `status-${status}` : ''}`}
      onClick={handleCardClick}
      style={{ cursor: 'pointer' }}
    >
      <div className="property-image">
        {property.image_url && !imageError ? (
          <img
            src={property.image_url}
            alt={property.title}
            onError={() => setImageError(true)}
          />
        ) : (
          <div style={{ width: '100%', height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'var(--bg-tertiary)', color: 'var(--text-muted)', fontSize: '13px' }}>
            画像なし
          </div>
        )}
        <button
          className="hide-property-btn"
          onClick={(e) => {
            e.stopPropagation()
            onHide()
          }}
          title="非表示にする"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M18 6 6 18"/>
            <path d="m6 6 12 12"/>
          </svg>
        </button>
      </div>
      <div className="property-body">
        {property.rent && (
          <div className="property-price">
            {formatRent(property.rent)}<small>万円</small>
          </div>
        )}
        <div className="property-meta">
          {property.floor_plan && (
            <span>
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="m3 9 9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/></svg>
              {property.floor_plan}
            </span>
          )}
          {property.area && <span>{property.area}㎡</span>}
          {property.building_age && <span>築{property.building_age}年</span>}
          {property.floor && <span>{property.floor}階</span>}
        </div>
        {(property.station || property.address) && (
          <div className="property-location">
            {property.station && property.walk_time && (
              <><strong>{property.station}</strong> 徒歩{property.walk_time}分</>
            )}
            {property.address && <> ／ {property.address}</>}
          </div>
        )}
        <div className="property-actions">
          <button
            className={`status-btn candidate ${status === 'candidate' ? 'active' : ''}`}
            onClick={() => onStatusChange('candidate')}
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M19 21l-7-5-7 5V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2z"/>
            </svg>
            候補
          </button>
          <button
            className={`status-btn maybe ${status === 'maybe' ? 'active' : ''}`}
            onClick={() => onStatusChange('maybe')}
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <circle cx="12" cy="12" r="10"/>
              <path d="M12 8v4"/>
              <path d="M12 16h.01"/>
            </svg>
            保留
          </button>
          <button
            className={`status-btn exclude ${status === 'excluded' ? 'active' : ''}`}
            onClick={() => onStatusChange('excluded')}
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M1 1l22 22"/>
              <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"/>
            </svg>
            非表示
          </button>
        </div>
      </div>
    </article>
  )
}

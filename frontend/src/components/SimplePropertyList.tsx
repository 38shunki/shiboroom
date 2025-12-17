'use client'

import { useState, useEffect } from 'react'

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8084'

interface Property {
  id: string
  title: string
  rent: number
  floor_plan?: string
  area?: number
  walk_time?: number
  building_age?: number
  image_url?: string
}

export default function SimplePropertyList() {
  const [properties, setProperties] = useState<Property[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Use useEffect with explicit dependency
  useEffect(() => {
    let mounted = true

    const fetchData = async () => {
      try {
        console.log('🔄 Fetching from:', `${API_URL}/api/properties?limit=1000`)
        const response = await fetch(`${API_URL}/api/properties?limit=1000`, {
          cache: 'no-store',
          headers: {
            'Content-Type': 'application/json',
          },
        })

        if (!response.ok) {
          throw new Error(`HTTP error! status: ${response.status}`)
        }

        const data = await response.json()

        // API returns array directly, not wrapped in {properties: [...]}
        const propertiesArray = Array.isArray(data) ? data : (data.properties || [])
        console.log('✅ Received:', propertiesArray.length, 'properties')

        if (mounted) {
          setProperties(propertiesArray)
          setLoading(false)
        }
      } catch (err) {
        console.error('❌ Error:', err)
        if (mounted) {
          setError(err instanceof Error ? err.message : 'Unknown error')
          setLoading(false)
        }
      }
    }

    fetchData()

    return () => {
      mounted = false
    }
  }, [])

  if (loading) {
    return (
      <div style={{ padding: '40px', textAlign: 'center' }}>
        <p>読み込み中...</p>
      </div>
    )
  }

  if (error) {
    return (
      <div style={{ padding: '40px', textAlign: 'center', color: 'red' }}>
        <p>エラー: {error}</p>
      </div>
    )
  }

  return (
    <div style={{ padding: '20px' }}>
      <h1>物件一覧 ({properties.length}件)</h1>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))', gap: '20px' }}>
        {properties.map(property => (
          <div key={property.id} style={{ border: '1px solid #ddd', padding: '15px', borderRadius: '8px' }}>
            {property.image_url && (
              <img
                src={property.image_url}
                alt={property.title}
                style={{ width: '100%', height: '200px', objectFit: 'cover', borderRadius: '4px' }}
              />
            )}
            <h3 style={{ fontSize: '14px', margin: '10px 0' }}>{property.title}</h3>
            <p><strong>{property.rent ? (property.rent / 10000).toFixed(1) : '?'}万円</strong></p>
            {property.floor_plan && <p>間取り: {property.floor_plan}</p>}
            {property.area && <p>面積: {property.area}㎡</p>}
            {property.walk_time && <p>徒歩: {property.walk_time}分</p>}
          </div>
        ))}
      </div>
    </div>
  )
}

"use client";

// TAREA: "dashboard interactivo con mapa en tiempo real (MapLibre)".
import { useEffect, useRef } from "react";
import { Map as MapLibreMap, Marker, NavigationControl, LngLatBounds, type StyleSpecification } from "maplibre-gl";
import "maplibre-gl/dist/maplibre-gl.css";
import type { Vehicle } from "@/types/api";
import { LOW_FUEL_THRESHOLD_HOURS } from "@/types/api";

interface VehicleMapProps {
  vehicles: Vehicle[];
  selectedId?: string | null;
  onSelect?: (id: string) => void;
}

// El estilo demo vectorial de MapLibre (demotiles.maplibre.org) solo trae
// polígonos de países a baja resolución (maxzoom 6): a nivel de ciudad se ve
// un rectángulo de color liso, sin calles ni vías, así que no se distingue
// si un vehículo está en zona urbana o en carretera. En su lugar usamos los
// tiles raster de OpenStreetMap (sin API key) que sí traen esa resolución.
const MAP_STYLE: StyleSpecification = {
  version: 8,
  sources: {
    osm: {
      type: "raster",
      tiles: ["https://tile.openstreetmap.org/{z}/{x}/{y}.png"],
      tileSize: 256,
      attribution: "© OpenStreetMap contributors",
      maxzoom: 19,
    },
  },
  layers: [{ id: "osm", type: "raster", source: "osm" }],
};

/** Mapa en tiempo real reutilizable: un marker por vehículo con lectura
 * GPS, coloreado por estado (rojo = combustible bajo <1h de autonomía). */
export function VehicleMap({ vehicles, selectedId, onSelect }: VehicleMapProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const mapRef = useRef<MapLibreMap | null>(null);
  const markersRef = useRef<globalThis.Map<string, Marker>>(new globalThis.Map());
  const hasFitBoundsRef = useRef(false);
  const prevSelectedIdRef = useRef<string | null>(null);

  useEffect(() => {
    if (!containerRef.current || mapRef.current) return;

    const map = new MapLibreMap({
      container: containerRef.current,
      style: MAP_STYLE,
      center: [0, 0],
      zoom: 2,
    });
    map.addControl(new NavigationControl(), "top-right");
    mapRef.current = map;

    return () => {
      map.remove();
      mapRef.current = null;
    };
  }, []);

  useEffect(() => {
    const map = mapRef.current;
    if (!map) return;

    const withCoords = vehicles.filter((v) => v.latest_reading);
    const seenIds = new Set(withCoords.map((v) => v.id));

    for (const [id, marker] of markersRef.current) {
      if (!seenIds.has(id)) {
        marker.remove();
        markersRef.current.delete(id);
      }
    }

    for (const v of withCoords) {
      const reading = v.latest_reading!;
      const lngLat: [number, number] = [reading.lon, reading.lat];
      // TAREA: colorea en rojo los vehículos con alerta predictiva de combustible.
      const isLowFuel =
        reading.autonomy_hours !== undefined && reading.autonomy_hours < LOW_FUEL_THRESHOLD_HOURS;

      let marker = markersRef.current.get(v.id);
      if (!marker) {
        const el = document.createElement("button");
        el.type = "button";
        el.setAttribute("aria-label", `Vehículo ${v.id}`);
        el.className = "h-4 w-4 rounded-full border-2 border-white shadow-md cursor-pointer";
        el.onclick = () => onSelect?.(v.id);
        marker = new Marker({ element: el }).setLngLat(lngLat).addTo(map);
        markersRef.current.set(v.id, marker);
      } else {
        marker.setLngLat(lngLat);
      }

      marker.getElement().style.backgroundColor = isLowFuel
        ? "#ef4444"
        : v.id === selectedId
          ? "#0ea5e9"
          : "#16a34a";
    }

    // TAREA (mejora UX): al elegir un vehículo desde la lista, centrar el
    // mapa en su posición — antes el mapa se quedaba quieto y con flotas
    // dispersas geográficamente el admin tenía que buscar a mano el punto
    // resaltado entre todos los marcadores.
    if (selectedId && selectedId !== prevSelectedIdRef.current) {
      const selected = withCoords.find((v) => v.id === selectedId);
      if (selected) {
        map.flyTo({ center: [selected.latest_reading!.lon, selected.latest_reading!.lat], zoom: Math.max(map.getZoom(), 11), duration: 600 });
      }
    }
    prevSelectedIdRef.current = selectedId ?? null;

    if (withCoords.length > 0 && !hasFitBoundsRef.current) {
      // Con un único punto (un solo vehículo, o varios en la misma
      // coordenada), LngLatBounds queda con área cero: fitBounds() sobre un
      // bounds degenerado deja el mapa "roto" (no vuelve a pedir tiles,
      // el canvas queda en blanco). En ese caso centramos con jumpTo en vez
      // de fitBounds.
      const first: [number, number] = [withCoords[0].latest_reading!.lon, withCoords[0].latest_reading!.lat];
      const isSinglePoint = withCoords.every(
        (v) => v.latest_reading!.lon === first[0] && v.latest_reading!.lat === first[1],
      );

      if (isSinglePoint) {
        map.jumpTo({ center: first, zoom: 12 });
      } else {
        const bounds = new LngLatBounds();
        withCoords.forEach((v) => bounds.extend([v.latest_reading!.lon, v.latest_reading!.lat]));
        map.fitBounds(bounds, { padding: 60, maxZoom: 12, duration: 0 });
      }
      hasFitBoundsRef.current = true;
    }
  }, [vehicles, selectedId, onSelect]);

  return <div ref={containerRef} className="h-full min-h-[320px] w-full rounded-lg" />;
}

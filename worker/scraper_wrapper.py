#!/usr/bin/env python3
"""
Scraper wrapper that outputs JSON for Go worker integration.
This wraps the ground-reservation scrapers and outputs structured JSON.
"""

import json
import os
import re
import sys
from datetime import datetime
from typing import Dict, Any

# Configure all scrapers with wide time range (00:00 - 23:59) and all weekdays
ALL_WEEKDAYS = "月曜日,火曜日,水曜日,木曜日,金曜日,土曜日,日曜日,祝日"
for prefix in ("HIRATSUKA", "AYASE", "YOKOHAMA"):
    os.environ.setdefault(f"{prefix}_TIME_FROM", "0000")
    os.environ.setdefault(f"{prefix}_TIME_TO", "2359")
    os.environ.setdefault(f"{prefix}_SELECTED_WEEK_DAYS", ALL_WEEKDAYS)

script_dir = os.path.dirname(os.path.abspath(__file__))
ground_reservation_path = os.path.join(script_dir, '..', '..', 'ground-reservation')
sys.path.insert(0, ground_reservation_path)

try:
    from app.facilities.yokohama.yokohama_facility import YokohamaFacility
    from app.facilities.ayase.ayase_facility import AyaseFacility
    from app.facilities.hiratsuka.hiratsuka_facility import HiratsukaFacility
    from app.facilities.kanagawa_system.kanagawa.kanagawa_facility import KanagawaFacility
    from app.facilities.kanagawa_system.kamakura.kamakura_facility import KamakuraFacility
    from app.facilities.kanagawa_system.fujisawa.fujisawa_facility import FujisawaFacility

    SCRAPERS_AVAILABLE = True
except ImportError:
    SCRAPERS_AVAILABLE = False


def _make_slot(date: str, time_from: str, time_to: str, court_name: str,
                raw_text: str, facility_type: str) -> Dict[str, Any]:
    """Create a slot dict with parsed values."""
    return {
        "date": date,
        "time_from": time_from,
        "time_to": time_to,
        "court_name": court_name.strip() if court_name else None,
        "raw_text": raw_text,
        "facility_type": facility_type
    }


def _infer_year(month: int) -> int:
    """Infer year from month - if month is past, assume next year."""
    now = datetime.now()
    return now.year + 1 if month < now.month else now.year


def parse_slot_string(slot_str: str, facility_type: str) -> Dict[str, Any]:
    """Parse slot string into structured dict. Supports multiple date formats."""
    slot_str = slot_str.strip()

    # Japanese era format: "令和08年02月28日(土) 06:00 ～ 08:00 施設名"
    if match := re.match(r'令和(\d+)年(\d+)月(\d+)日\([^)]+\)\s+(\d+:\d+)\s*[～~-]\s*(\d+:\d+)\s+(.+)', slot_str):
        era_year, month, day, time_from, time_to, court_name = match.groups()
        year = 2018 + int(era_year)  # 令和元年 = 2019
        return _make_slot(f"{year}-{int(month):02d}-{int(day):02d}",
                          time_from, time_to, court_name, slot_str, facility_type)

    # Slash format: "01/17(土) 13:00 ～ 15:00 施設名"
    if match := re.match(r'(\d+)/(\d+)\([^)]+\)\s+(\d+:\d+)\s*[～~-]\s*(\d+:\d+)\s+(.+)', slot_str):
        month, day, time_from, time_to, court_name = match.groups()
        year = _infer_year(int(month))
        return _make_slot(f"{year}-{int(month):02d}-{int(day):02d}",
                          time_from, time_to, court_name, slot_str, facility_type)

    # Kanji format: "1月15日(土) 09:00-12:00 施設名"
    if match := re.match(r'(\d+)月(\d+)日\([^)]+\)\s+(\d+:\d+)\s*[～~-]\s*(\d+:\d+)\s+(.+)', slot_str):
        month, day, time_from, time_to, court_name = match.groups()
        year = _infer_year(int(month))
        return _make_slot(f"{year}-{int(month):02d}-{int(day):02d}",
                          time_from, time_to, court_name, slot_str, facility_type)

    # ISO format: "2024-01-15 09:00-12:00 施設名"
    if match := re.match(r'(\d{4}-\d{2}-\d{2})\s+(\d+:\d+)-(\d+:\d+)\s+(.+)', slot_str):
        date, time_from, time_to, court_name = match.groups()
        return _make_slot(date, time_from, time_to, court_name, slot_str, facility_type)

    # Fallback: unparseable
    return _make_slot(None, None, None, None, slot_str, facility_type)


def search_facility(facility_type: str) -> Dict[str, Any]:
    """
    Search a specific facility and return structured results.
    """
    if not SCRAPERS_AVAILABLE:
        return {
            "success": False,
            "error": "Scrapers not available. Install ground-reservation.",
            "facility_type": facility_type,
            "slots": []
        }
    
    scrapers = {
        "yokohama": YokohamaFacility,
        "ayase": AyaseFacility,
        "hiratsuka": HiratsukaFacility,
        "kanagawa": KanagawaFacility,
        "kamakura": KamakuraFacility,
        "fujisawa": FujisawaFacility,
    }
    
    if facility_type not in scrapers:
        return {
            "success": False,
            "error": f"Unknown facility type: {facility_type}",
            "facility_type": facility_type,
            "slots": []
        }
    
    try:
        facility = scrapers[facility_type]()
        raw_results = facility.search_facility()
        
        slots = []
        for slot_str in raw_results:
            if slot_str:  # Skip empty strings
                parsed = parse_slot_string(slot_str, facility_type)
                slots.append(parsed)
        
        return {
            "success": True,
            "error": None,
            "facility_type": facility_type,
            "slots": slots,
            "scraped_at": datetime.now().isoformat()
        }
    except Exception as e:
        return {
            "success": False,
            "error": str(e),
            "facility_type": facility_type,
            "slots": []
        }


def main():
    if len(sys.argv) < 2:
        print(json.dumps({
            "success": False,
            "error": "Usage: scraper_wrapper.py <facility_type>",
            "available_types": ["yokohama", "ayase", "hiratsuka", "kanagawa", "kamakura", "fujisawa"]
        }))
        sys.exit(1)
    
    facility_type = sys.argv[1]
    result = search_facility(facility_type)
    print(json.dumps(result, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()

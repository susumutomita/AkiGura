#!/usr/bin/env python3
"""
Scraper wrapper that outputs JSON for Go worker integration.
This wraps the ground-reservation scrapers and outputs structured JSON.
"""

import json
import sys
import re
from datetime import datetime
from typing import List, Dict, Any

# Add ground-reservation to path
import os

# Set environment variables for wide time range BEFORE importing scrapers
# This allows all time slots to be returned (00:00 - 23:59)
os.environ.setdefault("HIRATSUKA_TIME_FROM", "0000")
os.environ.setdefault("HIRATSUKA_TIME_TO", "2359")
os.environ.setdefault("AYASE_TIME_FROM", "0000")
os.environ.setdefault("AYASE_TIME_TO", "2359")
os.environ.setdefault("YOKOHAMA_TIME_FROM", "0000")
os.environ.setdefault("YOKOHAMA_TIME_TO", "2359")
# Search all days of the week (Japanese weekday names)
os.environ.setdefault("HIRATSUKA_SELECTED_WEEK_DAYS", "月曜日,火曜日,水曜日,木曜日,金曜日,土曜日,日曜日,祝日")
os.environ.setdefault("AYASE_SELECTED_WEEK_DAYS", "月曜日,火曜日,水曜日,木曜日,金曜日,土曜日,日曜日,祝日")
os.environ.setdefault("YOKOHAMA_SELECTED_WEEK_DAYS", "月曜日,火曜日,水曜日,木曜日,金曜日,土曜日,日曜日,祝日")

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


def parse_slot_string(slot_str: str, facility_type: str) -> Dict[str, Any]:
    """
    Parse a slot string like "\n2024-01-15 09:00-12:00 新横浜公園野球場"
    into a structured dict.
    """
    slot_str = slot_str.strip()

    # Try to parse Japanese era date format: "令和08年02月28日(土) 06:00 ～ 08:00 施設名"
    jp_era_pattern = r'令和(\d+)年(\d+)月(\d+)日\([^)]+\)\s+(\d+:\d+)\s*[～~-]\s*(\d+:\d+)\s+(.+)'
    match = re.match(jp_era_pattern, slot_str)
    if match:
        era_year, month, day, time_from, time_to, court_name = match.groups()
        # 令和元年 = 2019, so 令和N年 = 2018 + N
        year = 2018 + int(era_year)
        return {
            "date": f"{year}-{int(month):02d}-{int(day):02d}",
            "time_from": time_from,
            "time_to": time_to,
            "court_name": court_name.strip(),
            "raw_text": slot_str,
            "facility_type": facility_type
        }

    # Try to parse Japanese date format: "01/17(土) 13:00 ～ 15:00 施設名"
    jp_slash_pattern = r'(\d+)/(\d+)\([^)]+\)\s+(\d+:\d+)\s*[～~-]\s*(\d+:\d+)\s+(.+)'
    match = re.match(jp_slash_pattern, slot_str)
    if match:
        month, day, time_from, time_to, court_name = match.groups()
        year = datetime.now().year
        # If the month is less than current month, it's next year
        if int(month) < datetime.now().month:
            year += 1
        return {
            "date": f"{year}-{int(month):02d}-{int(day):02d}",
            "time_from": time_from,
            "time_to": time_to,
            "court_name": court_name.strip(),
            "raw_text": slot_str,
            "facility_type": facility_type
        }
    
    # Try to parse Japanese date format: "1月15日(土) 09:00-12:00 ..."
    jp_pattern = r'(\d+)月(\d+)日\([^)]+\)\s+(\d+:\d+)\s*[～~-]\s*(\d+:\d+)\s+(.+)'
    match = re.match(jp_pattern, slot_str)
    if match:
        month, day, time_from, time_to, court_name = match.groups()
        year = datetime.now().year
        # If the month is less than current month, it's next year
        if int(month) < datetime.now().month:
            year += 1
        return {
            "date": f"{year}-{int(month):02d}-{int(day):02d}",
            "time_from": time_from,
            "time_to": time_to,
            "court_name": court_name.strip(),
            "raw_text": slot_str,
            "facility_type": facility_type
        }
    
    # Try ISO date format: "2024-01-15 09:00-12:00 ..."
    iso_pattern = r'(\d{4}-\d{2}-\d{2})\s+(\d+:\d+)-(\d+:\d+)\s+(.+)'
    match = re.match(iso_pattern, slot_str)
    if match:
        date, time_from, time_to, court_name = match.groups()
        return {
            "date": date,
            "time_from": time_from,
            "time_to": time_to,
            "court_name": court_name.strip(),
            "raw_text": slot_str,
            "facility_type": facility_type
        }
    
    # Fallback: return raw string
    return {
        "date": None,
        "time_from": None,
        "time_to": None,
        "court_name": None,
        "raw_text": slot_str,
        "facility_type": facility_type
    }


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

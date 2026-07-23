import csv
import sys

with open('deployments/seed_data/public.mitsubishi_tag_lists.csv', 'r') as f_in, open('tags_filtered.csv', 'w') as f_out:
    reader = csv.DictReader(f_in)
    
    # columns for the struct: id, created_at, updated_at, deleted_at, facility_name, robot_id, plc_id, tag_name, tag_address, comment, data_type, action, screen, svg_element, true_condition_color, false_condition_color, blinking, refresh_rate
    # Wait, deleted_at in Gorm is a timestamp.
    fieldnames = ['id', 'created_at', 'updated_at', 'deleted_at', 'facility_name', 'robot_id', 'plc_id', 'tag_name', 'tag_address', 'comment', 'data_type', 'action', 'screen', 'svg_element', 'true_condition_color', 'false_condition_color', 'blinking', 'refresh_rate']
    
    writer = csv.DictWriter(f_out, fieldnames=fieldnames)
    writer.writeheader()
    
    for row in reader:
        out_row = {}
        for col in fieldnames:
            val = row.get(col, '')
            if val == '' and col == 'deleted_at':
                out_row[col] = None
            elif col == 'refresh_rate' and val == '':
                out_row[col] = 0
            else:
                out_row[col] = val
        writer.writerow(out_row)
